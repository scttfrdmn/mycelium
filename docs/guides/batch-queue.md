---
description: "A batch queue runs a sequence of dependent jobs on a single instance, automatically chaining them in order."
---

# Batch Job Queues

A batch queue runs a sequence of dependent jobs on a single instance, automatically chaining them in order. Unlike [pipelines](/guides/pipelines), which chain stages across separate instances, a batch queue runs all jobs on the same instance — useful when jobs need to share local data or a common environment.

## When to use a batch queue

- Jobs that share large local datasets (avoid re-staging between instances)
- Sequential steps where each step depends on the previous one's output
- Workflows where spinning up a new instance per step is too slow or costly

For jobs that can run in parallel or need different instance types per step, use [pipelines](/guides/pipelines) or [parameter sweeps](/guides/parameter-sweeps) instead.

## Queue file format

Create a JSON file describing your jobs:

```json
{
  "queue_name": "my-pipeline",
  "jobs": [
    {
      "job_id": "preprocess",
      "command": "python preprocess.py --input /data/raw --output /data/clean",
      "timeout": "30m"
    },
    {
      "job_id": "train",
      "command": "python train.py --data /data/clean --output /models/",
      "timeout": "4h",
      "depends_on": ["preprocess"]
    },
    {
      "job_id": "evaluate",
      "command": "python eval.py --model /models/best.pt",
      "timeout": "30m",
      "depends_on": ["train"]
    }
  ],
  "global_timeout": "6h",
  "on_failure": "stop"
}
```

**Fields:**

| Field | Required | Description |
|-------|----------|-------------|
| `queue_name` | yes | Human-readable name |
| `jobs` | yes | Array of job definitions |
| `job_id` | yes | Unique identifier for this job |
| `command` | yes | Shell command to run |
| `timeout` | no | Per-job timeout (e.g. `30m`, `4h`) |
| `depends_on` | no | List of `job_id`s that must complete first |
| `global_timeout` | no | Total timeout for the entire queue |
| `on_failure` | no | `stop` (default) or `continue` on job failure |

## Launch

```sh
spawn launch my-job \
  --instance-type g5.xlarge \
  --batch-queue pipeline.json \
  --ttl 8h
```

spawn uploads the queue config to S3, launches the instance, and the spored daemon picks up and executes the jobs sequentially according to the dependency graph.

## Monitoring

```sh
# Check instance status
spawn status my-job

# Connect and watch logs
spawn connect my-job -- tail -f /var/log/spored.log

# The instance auto-terminates (or stops) when the queue completes
```

## Example: ML training pipeline

```json
{
  "queue_name": "classifier-pipeline",
  "jobs": [
    {
      "job_id": "download",
      "command": "aws s3 cp s3://my-bucket/data/ /data/ --recursive",
      "timeout": "20m"
    },
    {
      "job_id": "preprocess",
      "command": "python preprocess.py",
      "timeout": "1h",
      "depends_on": ["download"]
    },
    {
      "job_id": "train",
      "command": "python train.py --epochs 50",
      "timeout": "6h",
      "depends_on": ["preprocess"]
    },
    {
      "job_id": "upload-results",
      "command": "aws s3 cp /models/ s3://my-bucket/models/ --recursive",
      "timeout": "10m",
      "depends_on": ["train"]
    }
  ],
  "global_timeout": "8h",
  "on_failure": "stop"
}
```

```sh
spawn launch classifier \
  --instance-type g5.xlarge \
  --spot \
  --batch-queue pipeline.json \
  --ttl 10h
```
