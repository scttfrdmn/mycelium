---
description: "A parameter sweep runs the same job across many combinations of input parameters, each on its own instance, in parallel."
---

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

You enumerate the combinations you want in `params` — spawn does not expand a grid
for you. To sweep the full grid of several lists, generate the `params` list with a
few lines of your own script (any language) and write it to the YAML/JSON file:

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
Passing parameters inline (`--params …`) and auto-generating ranges/cartesian products from the CLI are not yet available — the CLI returns *"inline --params not yet implemented, use --param-file for now."* Generate the `params` list into a file as shown above. Native `grid:`/matrix expansion is tracked in [spawn#390](https://github.com/spore-host/spawn/issues/390).
:::

## Heterogeneous sweeps — vary the instance type per entry

A sweep entry can set its own `instance_type` (and `ami`, `spot`, `region`, `az`), so a single sweep can run the **same workload across different instance families** — the natural shape of a price-performance benchmark. Any field an entry sets overrides the top-level `--instance-type` / `defaults` for that entry only; entries that omit `instance_type` fall back to the CLI `--instance-type`.

```yaml
# gromacs-bench.yaml — one workload, many instance types, compare ns/$
defaults:
  ttl: 2h
  on_complete: terminate
  spot: true
  command: "gmx mdrun -s bench.tpr && aws s3 cp md.log s3://my-bucket/bench/{instance_type}/"

params:
  - instance_type: c8i.24xlarge   # Intel
  - instance_type: c8a.24xlarge   # AMD
  - instance_type: c8g.24xlarge   # Graviton (arm64)
  - instance_type: g6.2xlarge     # NVIDIA L4 GPU
  - instance_type: g6e.2xlarge    # NVIDIA L40S GPU
```

```sh
spawn launch gromacs-bench --param-file gromacs-bench.yaml
```

spawn detects the right AMI **per entry** from its instance type — an arm64 AMI for `c8g`, a GPU AMI for `g6`/`g6e`, an x86 AMI for `c8i`/`c8a` — so you don't hand-pick an AMI per family (entries sharing an architecture reuse one AMI lookup). Set an explicit `ami:` on an entry to override. Each instance uploads its result keyed by `{instance_type}`, so the comparison falls out of the S3 layout.

::: warning One OS per sweep
A sweep must be all-Linux or all-Windows — a single `command`/lifecycle model can't span both. Mixing an entry that resolves to Windows with Linux entries is rejected before launch.
:::

::: info Detached (Lambda) sweeps
`--detach` sweeps run through the Lambda orchestrator, which uses each entry's explicit `ami:` but does **not** auto-detect one. For a heterogeneous `--detach` sweep, set `ami:` on every entry.
:::

## Monitoring a sweep

```sh
spawn list --sweep-name hp-search     # all instances in the sweep
spawn sweep status <sweep-id>         # summary: running, completed, failed
spawn sweep cancel <sweep-id>         # terminate all remaining instances
```

With Slack connected, you'll get a DM when the sweep finishes (all instances have terminated).

## Controlling concurrency and cost

A large sweep does not have to launch every instance at once. Two launch flags cap
how wide and how expensive it gets:

```sh
spawn launch hp-search --param-file sweep.yaml \
  --max-concurrent 8 \      # at most 8 instances running at a time (0 = unlimited)
  --budget 200 \            # stop launching once projected spend hits $200 (0 = no limit)
  --ttl 4h
```

`--max-concurrent` (or `--max-concurrent-per-region`) queues the remaining entries
and starts them as running ones finish — useful for staying under an instance-family
quota. `--budget` is a spend ceiling for the whole sweep.

## Resuming an interrupted sweep

Sweeps checkpoint their progress, so a sweep that was cancelled or partially
launched can be resumed without re-running the entries that already completed:

```sh
spawn sweep resume <sweep-id>                     # continue from checkpoint
spawn sweep resume <sweep-id> --max-concurrent 5  # optionally re-cap concurrency
```

Gather the results of a finished sweep into one place with:

```sh
spawn sweep collect <sweep-id> --output results.json
```

::: info Re-running only the failed entries
There isn't yet a one-flag "retry only the failed entries" for sweeps; `resume`
continues incomplete work from the checkpoint. First-class failed-subset rerun is
part of the array/sweep reporting work in [spawn#389](https://github.com/spore-host/spawn/issues/389).
:::

## Alerts on completion, failure, or cost

For explicit, per-sweep notifications — beyond the default Slack DM — attach an
**alert** to a sweep with `spawn alerts`. Alerts can fire on completion, on
failure, or when the sweep's running cost crosses a threshold, and deliver via
email, Slack, SNS, or a webhook:

```sh
spawn alerts create <sweep-id> --on-complete --email me@example.com
spawn alerts create <sweep-id> --on-failure  --slack https://hooks.slack.com/services/...
spawn alerts create <sweep-id> --cost-threshold 100 --email me@example.com

spawn alerts list                 # all alerts
spawn alerts history              # what has fired
spawn alerts delete <alert-id>
```

The cost-threshold alert is the cheap insurance for a large sweep: get pinged the
moment spend crosses your ceiling, then `spawn sweep cancel` if it's running away.

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
