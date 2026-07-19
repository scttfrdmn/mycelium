---
description: "nf-spawn is a Nextflow executor plugin that dispatches each pipeline process step to its own ephemeral EC2 instance — purpose-sized and auto-terminated when…"
---

# Nextflow Adapter (nf-spawn) <span class="doc-badge experimental">Experimental</span>

[nf-spawn](https://github.com/spore-host/nf-spawn) is a Nextflow executor plugin that dispatches each pipeline process step to its own ephemeral EC2 instance — purpose-sized and auto-terminated when the task completes.

This lets you run bioinformatics pipelines (including [nf-core](https://nf-co.re) pipelines) without AWS Batch, ECS, or any queue infrastructure.

::: warning Early prototype
nf-spawn is an **early prototype — not production-ready** (per its
[README](https://github.com/spore-host/nf-spawn)). It's one of five
[workflow adapters](/guides/workflow-engines), each early-stage; see the
[maturity matrix](/guides/workflow-engines#maturity-compatibility). Install today
requires cloning and building the plugin (below), not a drop-in registry install.
Validate against your own pipeline before relying on it.
:::

## How it works

```
Nextflow process → spawn launch nf-{hash} --instance-type <type> --on-complete terminate
                → instance runs task script
                → spored signals completion
             ← spawn status --check-complete polls until done
```

Each task gets a fresh instance. When the task script finishes, spored terminates the instance automatically.

## Requirements

- [spawn](https://github.com/spore-host/spawn) and [truffle](https://github.com/spore-host/truffle) on `PATH`
- AWS credentials configured
- Nextflow 26.04.x (the version nf-spawn v0.8.0 is built against)
- JDK 21+ (to build the plugin)

## Installation

```bash
# Clone and build+install into ~/.nextflow/plugins/
git clone https://github.com/spore-host/nf-spawn
cd nf-spawn
./gradlew installPlugin
```

See the [nf-spawn README](https://github.com/spore-host/nf-spawn) for the exact
plugin id/version to declare (it tracks the built version).

## Configuration

```groovy
// nextflow.config
plugins {
    id 'nf-spawn@0.8.0'
}

process {
    executor = 'spawn'

    // Default instance type for all processes
    ext.instanceType = 't3.medium'
    ext.region       = 'us-east-1'
    ext.ttl          = '2h'

    // Per-process overrides
    withName: 'KRAKEN2' {
        ext.instanceType = 'c7g.4xlarge'   // 16 vCPU, 32 GB RAM
        ext.spot         = true
    }
    withName: 'FASTP' {
        ext.instanceType = 't4g.medium'
    }
}

// S3 work directory (required — instances need shared storage)
workDir = 's3://my-bucket/nextflow-work'
```

## Example pipeline

```nextflow
process FASTP {
    input:  path reads
    output: path 'trimmed.fastq.gz'
    script:
    """
    fastp -i $reads -o trimmed.fastq.gz --thread ${task.cpus}
    """
}

process KRAKEN2 {
    input:  path reads
    output: path 'report.txt'
    script:
    """
    kraken2 --db /kraken2-db --report report.txt $reads
    """
}

workflow {
    reads = Channel.fromPath('s3://my-bucket/data/*.fastq.gz')
    FASTP(reads) | KRAKEN2
}
```

```bash
nextflow run my-pipeline.nf -profile aws
```

## Cost model

Each process step runs on its own instance, sized for that step. A typical nf-core/taxprofiler run with 100 samples:

- `fastp` QC: 100 × `t4g.medium` × ~3 min = ~$0.05 total
- `kraken2`: 100 × `c7g.4xlarge` × ~15 min = ~$5.80 total

No Batch job queue, no ECS cluster, no idle capacity — instances spin up and down per task.

## See also

- [Workflow Engines overview](/guides/workflow-engines) — all five engine integrations
- [nf-spawn on GitHub](https://github.com/spore-host/nf-spawn)
- [spawn launch reference](/tools/reference/spawn#spawn-launch)
- [Instance sizing with truffle](/tools/truffle)
