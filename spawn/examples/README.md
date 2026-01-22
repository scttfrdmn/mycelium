# spawn Examples

This directory contains example configurations for common spawn use cases.

## Directory Structure

- **`slurm/`** - Slurm batch script examples for HPC migration
- **`staging/`** - Data staging examples for multi-region distribution
- **`sweeps/`** - Parameter sweep examples for hyperparameter tuning
- **`mpi/`** - MPI job examples for parallel computing

## Quick Start

### Slurm Migration

1. **Convert existing Slurm script:**
```bash
spawn slurm convert examples/slurm/array-cpu.sbatch --output params.yaml
```

2. **Estimate cost:**
```bash
spawn slurm estimate examples/slurm/gpu-training.sbatch
```

3. **Submit job:**
```bash
spawn slurm submit examples/slurm/bioinformatics.sbatch --spot
```

### Data Staging

1. **Stage large dataset:**
```bash
# Example: 100GB reference genome
spawn stage upload hg38-reference.fasta \
  --regions us-east-1,us-west-2 \
  --dest /mnt/data/reference.fasta
```

2. **Launch with staged data:**
```bash
spawn launch --params examples/staging/reference-genome.yaml
```

3. **Cost savings:**
   - Without staging: $900 (100 instances × 100GB × $0.09/GB)
   - With staging: $2 (replication cost)
   - **Savings: 99.8%**

### Parameter Sweeps

1. **Launch hyperparameter search:**
```bash
spawn launch --params examples/sweeps/hyperparameter-search.yaml
```

2. **Monitor progress:**
```bash
spawn sweep status <sweep-id>
```

3. **Collect top results:**
```bash
spawn collect-results --sweep-id <sweep-id> \
  --metric accuracy \
  --best 5 \
  --output top5.csv
```

### MPI Jobs

1. **Launch MPI simulation:**
```bash
spawn launch --params examples/mpi/mpi-simulation.yaml --mpi
```

2. **Features:**
   - Automatic placement groups
   - EFA for ultra-low latency
   - Passwordless SSH between nodes
   - Automatic hostfile generation

## Example Descriptions

### Slurm Scripts

**`array-cpu.sbatch`**
- 100-task array job
- 4 CPUs, 8GB RAM per task
- 2-hour runtime
- Perfect for parameter sweeps

**`gpu-training.sbatch`**
- 2× V100 GPUs
- 64GB RAM, 16 CPUs
- 8-hour training
- PyTorch/TensorFlow ready

**`bioinformatics.sbatch`**
- 50-sample genome analysis
- BWA/samtools pipeline
- Reference genome staging
- 4-hour per sample

### Staging Examples

**`ml-training.yaml`**
- 200GB training dataset
- 30 instances across 3 regions
- $1,072 cost savings (99.3%)
- Distributed PyTorch training

**`reference-genome.yaml`**
- 100GB reference genome (hg38)
- 100 sample analysis
- $898 cost savings (99.8%)
- BWA alignment pipeline

### Sweep Examples

**`hyperparameter-search.yaml`**
- 48-trial grid search
- 4 learning rates × 3 batch sizes × 4 models
- Multi-region distribution
- Automatic result collection

### MPI Examples

**`mpi-simulation.yaml`**
- 16-node MPI job (576 processes)
- EFA-enabled for < 10μs latency
- Placement groups for co-location
- 24-hour simulation

## Cost Comparison

| Use Case | Cluster (Free) | spawn On-Demand | spawn Spot | Savings |
|----------|----------------|-----------------|------------|---------|
| 100-task array | 2-3 days queue | $332 (2h) | $100 (2h) | Time > Cost |
| GPU training | 1 week queue | $48 (8h) | $14 (8h) | Time > Cost |
| Genome analysis | 3 days queue | $440 (4h) | $132 (4h) | Time > Cost |
| MPI simulation | 2 weeks queue | $1,152 (24h) | N/A | Time > Cost |

## Best Practices

### When to Use Spot Instances

✅ **Use spot:**
- Array jobs (independent tasks)
- Short-duration tasks (< 2 hours)
- Fault-tolerant workloads
- Development/testing

❌ **Avoid spot:**
- Long-running tasks (> 4 hours)
- Tightly-coupled MPI (> 16 nodes)
- Time-critical deadlines
- Stateful applications

### When to Use Data Staging

✅ **Use staging when:**
- Dataset > 10GB
- Multiple regions
- > 5 instances per region
- Data doesn't change

**Break-even:** Just 1 instance per region!

### Instance Type Selection

| Workload | Instance Family | Example |
|----------|----------------|---------|
| CPU-intensive | C5, C6i | c5.4xlarge |
| Memory-intensive | R5, R6i | r5.2xlarge |
| GPU training | P3, P4d | p3.2xlarge |
| GPU inference | G4dn, G5 | g4dn.xlarge |
| MPI/HPC | C5n, C6gn | c5n.18xlarge |

## Getting Help

- **Documentation:** See [SLURM_GUIDE.md](../SLURM_GUIDE.md), [DATA_STAGING_GUIDE.md](../DATA_STAGING_GUIDE.md)
- **Troubleshooting:** See [TROUBLESHOOTING.md](../TROUBLESHOOTING.md)
- **Issues:** https://github.com/scttfrdmn/mycelium/issues

## Contributing Examples

Have a useful example? Submit a PR!

**Example template:**
```yaml
# Clear description of use case
sweep_name: my-example
count: 10
regions: [us-east-1]
instance_type: t3.medium

base_command: |
  #!/bin/bash
  # Well-commented commands
  echo "Example task"
```
