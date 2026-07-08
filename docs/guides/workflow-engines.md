# Workflow Engines

spore.host is a first-class execution backend for **five workflow engines**. In
every case the model is the same: each task/step/job/rule runs on its own
**purpose-sized, ephemeral EC2 instance** that auto-terminates when it finishes —
no cluster, no queue, no standing capacity. You keep writing your workflow in the
engine you already use; spawn just runs the work.

Each integration is a small, versioned, released package that plugs into the
engine's own extension point and reuses the same spawn machinery (truffle
auto-sizing, `--on-complete terminate` + TTL, a durable `.exitcode`-in-S3
completion signal).

## The five integrations

| Engine | Package | How you enable it | Repo |
|--------|---------|-------------------|------|
| **Nextflow** | `nf-spawn` | `executor = 'spawn'` in `nextflow.config` | [spore-host/nf-spawn](https://github.com/spore-host/nf-spawn) |
| **WDL** | `miniwdl-spawn` | `MINIWDL__SCHEDULER__CONTAINER_BACKEND=spawn` | [spore-host/miniwdl-spawn](https://github.com/spore-host/miniwdl-spawn) |
| **CWL** | `cwl-spawn` | `cwl-spawn workflow.cwl inputs.yml` | [spore-host/cwl-spawn](https://github.com/spore-host/cwl-spawn) |
| **Snakemake** | `snakemake-executor-plugin-spawn` | `snakemake --executor spawn` | [spore-host/snakemake-executor-plugin-spawn](https://github.com/spore-host/snakemake-executor-plugin-spawn) |
| **Apache Airflow** | `spawn-airflow` | `SpawnRunTaskOperator(...)` in a DAG | [spore-host/spawn-airflow](https://github.com/spore-host/spawn-airflow) |

The three [AWS HealthOmics](https://aws.amazon.com/healthomics/)-supported
languages — **Nextflow, WDL, CWL** — were prioritized first for life-sciences
relevance (spore.host is a cost-efficient alternative to HealthOmics, not a client
of it). **Snakemake** and **Airflow** followed on demand.

## Which one?

- **Already have a Nextflow / WDL / CWL / Snakemake workflow?** Use the matching
  plugin — your workflow runs unchanged; only the executor changes, so the engine
  still owns parsing, scheduling, scatter/gather, and output collection.
- **Bioinformatics / nf-core pipelines?** → **Nextflow** ([guide](/guides/nextflow)).
- **Prefer declarative per-task resources with auto-sizing?** WDL, CWL, and
  Snakemake all declare CPU/RAM, which spawn feeds to `truffle` to pick the
  cheapest fitting instance automatically.
- **Orchestrating a broader DAG (not just a bioinformatics pipeline)?** →
  **Airflow**: add a `SpawnRunTaskOperator` task wherever you want a step to run
  on an ephemeral instance. It's deferrable, so wide fan-out DAGs don't pin a
  worker slot per in-flight instance.

## Sizing

Where the engine declares resources, spawn sizes the instance automatically via
`truffle search --pick-first` (cheapest instance that fits):

- **Nextflow** — `ext.instanceType` (explicit) per process.
- **WDL** — `runtime { cpu, memory }` → auto-sized (or `spawn_instance_type`).
- **CWL** — `ResourceRequirement` (`coresMin`/`ramMin`) → auto-sized.
- **Snakemake** — `threads` + `resources: mem_mb` → auto-sized.
- **Airflow** — `cpus=` / `memory_gib=` on the operator → auto-sized (or
  `instance_type=`).

## Requirements (all engines)

- [spawn](https://github.com/spore-host/spawn) and
  [truffle](https://github.com/spore-host/truffle) on `PATH`
- AWS credentials configured
- An S3 location for the work/exit-code bridge (each engine's docs name the exact
  flag or env var)

Every task launches with a **TTL backstop** and `--on-complete terminate`, so a
run can't leak billable instances even if a step is interrupted.

## Not yet first-class

Snakemake and Airflow were promoted from the "integration pattern" tier on
demand. Others — Prefect, Argo Workflows, Dagster, Luigi, Temporal, AWS Step
Functions — currently have example patterns (spawn invoked as a launcher via
`spawn pipeline` / `spawn queue`) rather than a native plugin. If you need one of
those promoted to first-class, open an issue.

## See also

- [Nextflow integration guide](/guides/nextflow)
- [Pipelines](/guides/pipelines) — the engine-agnostic `spawn pipeline` / `spawn queue`
- [Instance sizing with truffle](/tools/truffle)
