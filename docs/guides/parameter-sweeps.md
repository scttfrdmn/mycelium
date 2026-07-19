# Parameter Sweeps

A parameter sweep runs the same job across many combinations of input parameters, each on its own instance, in parallel. It's useful for hyperparameter search, sensitivity analysis, and any scenario where you want to explore a parameter space without waiting for jobs to run sequentially.

## The basic pattern

A sweep is driven by a **parameter file** (`--param-file`, YAML/JSON/CSV): `defaults` shared by every instance, plus a `params` list where each entry is one combination to launch.

```yaml
# sweep.yaml
defaults:
  instance_type: g5.xlarge
  ttl: 4h
  on_complete: terminate
  command: "python train.py --lr {learning_rate} --batch {batch_size}"

params:
  - learning_rate: 0.001
    batch_size: 32
  - learning_rate: 0.001
    batch_size: 64
  - learning_rate: 0.01
    batch_size: 32
  - learning_rate: 0.01
    batch_size: 64
```

```sh
spawn launch hp-search --param-file sweep.yaml
```

This launches one instance per `params` entry (4 here), each running `command` with its combination substituted (`{learning_rate}`, `{batch_size}`). Each instance has its own TTL and terminates independently when done.

::: tip Preview before you launch
Add `--estimate-only` to see the instance count and cost estimate without launching anything.
:::

## Every combination of several lists

List each parameter's values under `defaults` isn't how it works — instead enumerate the combinations you want in `params`. To sweep the full grid of several lists, generate the `params` list with a few lines of your own script (any language) and write it to the YAML/JSON file:

```python
# gen-sweep.py — write every learning_rate × batch_size combination
import itertools, yaml
lrs, batches = [0.001, 0.01, 0.1], [32, 64, 128]
params = [{"learning_rate": lr, "batch_size": b} for lr, b in itertools.product(lrs, batches)]
yaml.safe_dump({"defaults": {"instance_type": "g5.xlarge", "ttl": "4h",
    "command": "python train.py --lr {learning_rate} --batch {batch_size}"},
    "params": params}, open("sweep.yaml", "w"))
```

```sh
python gen-sweep.py && spawn launch grid-search --param-file sweep.yaml   # 9 instances
```

::: info Inline `--params` / auto-expanded ranges
Passing parameters inline (`--params …`) and auto-generating ranges/cartesian products from the CLI are not yet available — the CLI returns *"inline --params not yet implemented, use --param-file for now."* Generate the `params` list into a file as shown above.
:::

## Monitoring a sweep

```sh
spawn list --sweep-name hp-search     # all instances in the sweep
spawn sweep status <sweep-id>         # summary: running, completed, failed
spawn sweep cancel <sweep-id>         # terminate all remaining instances
```

With Slack connected, you'll get a DM when the sweep finishes (all instances have terminated).

## Collecting results

Each instance writes its results to a path you control — typically S3. The convention is to include the sweep index or parameters in the path:

```sh
spawn launch hp-search \
  --instance-type g5.xlarge \
  --param-file sweep.yaml \
  --command "python train.py --lr {learning_rate} && \
             aws s3 cp results.json s3://my-bucket/sweeps/hp-search/{index}/results.json && \
             touch /tmp/SPAWN_COMPLETE" \
  --on-complete terminate
```

Each instance has `{index}` (0-based position in the sweep) and all parameter values available as environment variables and template substitutions.

## Cost estimation

Before launching a large sweep, use `--estimate-only` to see the maximum possible cost without launching anything:

```sh
spawn launch hp-search \
  --instance-type g5.xlarge \
  --param-file sweep.yaml \
  --ttl 4h \
  --estimate-only
```

This shows the maximum cost if every instance runs for the full TTL. Actual cost is lower because most instances complete before the TTL.

## Next steps

- [Job Arrays](/guides/job-arrays) — for when you want a fixed count of identical instances rather than parameterised jobs
- [Pipelines](/guides/pipelines) — chain sweeps so stage 2 launches after stage 1 completes
