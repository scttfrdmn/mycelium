---
description: "spore.host provides native execution adapters for five workflow engines — run each task on its own ephemeral EC2 instance. All are early / experimental."
---

# Workflow Adapters

spore.host provides **native execution adapters** for five workflow engines. In
every case the model is the same: each task/step/job/rule runs on its own
**purpose-sized, ephemeral EC2 instance** that auto-terminates when it finishes —
no cluster, no queue, no standing capacity. You keep writing your workflow in the
engine you already use; spawn just runs the work.

Each adapter is a small, versioned package that plugs into the engine's own
extension point and reuses the same spawn machinery (truffle auto-sizing,
`--on-complete terminate` + TTL, a durable `.exitcode`-in-S3 completion signal).

::: warning These adapters are early — read the maturity matrix first
"Native adapter" means **we build and maintain it**; it does **not** mean
production-ready. All five are early-stage: two are pre-1.0 prototypes and three
are two-week-old `v0.1.0` initial releases. Validate against your own workflow and
budget before relying on one. The shared execution model (staging, retries,
completion) is still evolving — see [spawn#386](https://github.com/spore-host/spawn/issues/386).
:::

## Maturity & compatibility

| Engine (adapter) | Status | Latest | Real-AWS validation | Enable it with |
|------------------|--------|--------|---------------------|----------------|
| **Nextflow** (`nf-spawn`) | <span class="doc-badge experimental">Experimental</span> prototype — not production-ready | v0.8.0 | unit/integration | `executor = 'spawn'` in `nextflow.config` |
| **WDL** (`miniwdl-spawn`) | <span class="doc-badge experimental">Experimental</span> early | v0.1.0 | in progress ([#395](https://github.com/spore-host/spore-host/issues/395)) | `MINIWDL__SCHEDULER__CONTAINER_BACKEND=spawn` |
| **CWL** (`cwl-spawn`) | <span class="doc-badge experimental">Experimental</span> early (v0.1) | v0.1.0 | verified end-to-end, leak-checked | `cwl-spawn workflow.cwl inputs.yml` |
| **Snakemake** (`snakemake-executor-plugin-spawn`) | <span class="doc-badge experimental">Experimental</span> early (v0.1) | v0.1.0 | verified end-to-end, leak-checked | `snakemake --executor spawn` |
| **Apache Airflow** (`spawn-airflow`) | <span class="doc-badge experimental">Experimental</span> early (v0.1) | v0.1.0 | verified end-to-end, leak-checked | `SpawnRunTaskOperator(...)` in a DAG |

Repos: [nf-spawn](https://github.com/spore-host/nf-spawn) ·
[miniwdl-spawn](https://github.com/spore-host/miniwdl-spawn) ·
[cwl-spawn](https://github.com/spore-host/cwl-spawn) ·
[snakemake-executor-plugin-spawn](https://github.com/spore-host/snakemake-executor-plugin-spawn) ·
[spawn-airflow](https://github.com/spore-host/spawn-airflow). Each repo's README and
CHANGELOG are the authoritative status; the table above summarizes them.

The three [AWS HealthOmics](https://aws.amazon.com/healthomics/)-supported
languages — **Nextflow, WDL, CWL** — were prioritized first for life-sciences
relevance (spore.host is a cost-efficient alternative to HealthOmics, not a client
of it). **Snakemake** and **Airflow** followed on demand.

Not sure a workflow engine is the right layer at all? See
[Which execution tool?](/guides/choosing-execution).

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

## No native adapter yet

Snakemake and Airflow gained native adapters on demand. Others — Prefect, Argo
Workflows, Dagster, Luigi, Temporal, AWS Step Functions — currently have example
patterns (spawn invoked as a launcher via `spawn pipeline` / `spawn queue`) rather
than a native adapter. If you need one, open an issue.

## Building a new adapter

All five adapters reimplement the same machinery (resource→instance sizing, S3
staging, launch, polling, `.exitcode` interpretation, cancellation). A shared
task-execution protocol / adapter library is proposed in
[spawn#386](https://github.com/spore-host/spawn/issues/386) so new adapters
translate their native task object into one common spec rather than rebuilding it.
Start there if you're writing one.

## See also

- [Which execution tool?](/guides/choosing-execution) — adapter vs. sweep vs. pipeline vs. native engine
- [Nextflow adapter guide](/guides/nextflow)
- [Pipelines](/guides/pipelines) — the engine-agnostic `spawn pipeline` / `spawn queue`
- [Instance sizing with truffle](/tools/truffle)
