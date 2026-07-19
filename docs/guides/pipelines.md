# Pipelines

A pipeline chains stages together: when one stage completes, it automatically launches the next. This is useful for multi-step workflows — data preparation, training, evaluation, post-processing — where each stage has different compute requirements.

## The basic pattern

Each stage uses `--on-complete` to trigger the next stage. The completion file (`/tmp/SPAWN_COMPLETE`) is the handoff signal.

```sh
# Stage 1: data preparation (cheap CPU instance)
spawn launch \
  --name pipeline-prep \
  --instance-type c6i.4xlarge \
  --ttl 4h \
  --on-complete terminate \
  --completion-file /tmp/SPAWN_COMPLETE \
  --command "python prepare_data.py --output s3://my-bucket/prepared/ && touch /tmp/SPAWN_COMPLETE"
```

Stage 1 terminates when it writes the completion file. Stage 2 runs on whatever compute you want:

```sh
# Stage 2: training (GPU instance)
spawn launch \
  --name pipeline-train \
  --instance-type p4d.24xlarge \
  --ttl 24h \
  --on-complete terminate \
  --command "python train.py --data s3://my-bucket/prepared/ && touch /tmp/SPAWN_COMPLETE"
```

## Automated pipeline definition

Instead of launching stages by hand, define the whole DAG in a **pipeline definition file** and let a Lambda orchestrator manage the handoffs. `spawn pipeline launch` uploads the definition to S3 and starts the orchestrator, which launches each stage when its dependencies complete.

The definition is JSON. Stages declare their dependencies with `depends_on`, so the pipeline is a DAG — independent stages run in parallel, dependent ones wait:

```json
{
  "pipeline_id": "ml-pipeline",
  "pipeline_name": "Training pipeline",
  "on_failure": "stop",
  "stages": [
    {
      "stage_id": "prep",
      "instance_type": "c6i.4xlarge",
      "command": "python prepare_data.py --output s3://my-bucket/prepared/",
      "timeout": "4h"
    },
    {
      "stage_id": "train",
      "instance_type": "p4d.24xlarge",
      "spot": true,
      "depends_on": ["prep"],
      "command": "python train.py --data s3://my-bucket/prepared/",
      "timeout": "24h"
    },
    {
      "stage_id": "eval",
      "instance_type": "c6i.2xlarge",
      "depends_on": ["train"],
      "command": "python evaluate.py --model s3://my-bucket/model/",
      "timeout": "2h"
    }
  ]
}
```

Preview the DAG before running it, then launch:

```sh
spawn pipeline graph pipeline.json     # ASCII render of the dependency graph
spawn pipeline launch pipeline.json    # upload + start the orchestrator
spawn pipeline launch pipeline.json --wait   # block until the pipeline finishes
```

Each stage launches automatically once every stage in its `depends_on` list has succeeded. Set `"on_failure": "continue"` at the top level to keep running independent branches when one stage fails (the default, `"stop"`, halts the pipeline).

## Passing data between stages

Stages hand data off through S3. Give a stage a `data_output` block to publish its results, and a downstream stage a `data_input` block naming the source stage — the orchestrator wires the S3 locations so the consumer downloads them before its command runs:

```json
{
  "stage_id": "prep",
  "instance_type": "c6i.4xlarge",
  "command": "python prepare_data.py --output /mnt/out/",
  "data_output": { "mode": "s3", "paths": ["/mnt/out/"] }
},
{
  "stage_id": "train",
  "instance_type": "p4d.24xlarge",
  "depends_on": ["prep"],
  "data_input": { "mode": "s3", "source_stage": "prep", "dest_path": "/mnt/data" },
  "command": "python train.py --data /mnt/data"
}
```

For tightly-coupled stages that stream rather than stage through S3, use `"mode": "stream"` with a `protocol` (`tcp`, `grpc`, or `zmq`) and `port`; the orchestrator discovers peer addresses across stages.

## Error handling

If a stage exits non-zero the pipeline stops (unless `on_failure` is `continue`). Fix the cause and re-launch — re-running `spawn pipeline launch` starts a fresh run of the definition.

## Monitoring

`status`, `cancel`, and `collect` take the **pipeline id** (the `pipeline_id` from the definition); `graph` takes the definition **file**:

```sh
spawn pipeline list                    # all pipelines and their state
spawn pipeline status ml-pipeline      # current stage, elapsed time, cost
spawn pipeline collect ml-pipeline     # download results (--stage to pick one)
spawn pipeline cancel ml-pipeline      # stop the current stage and all remaining
```
