# Genomics Pipeline with BAMS3 + ZeroMQ + RDMA

**High-throughput variant calling pipeline using cloud-native genomics formats and RDMA networking.**

## Overview

This pipeline demonstrates how to combine:
- **BAMS3** - Cloud-native BAM format with independent chunks
- **ZeroMQ** - Message queue patterns for pipeline coordination
- **RDMA/EFA** - 100+ Gbps networking for massive throughput
- **Spawn pipelines** - Multi-stage orchestration

### Performance

**With BAMS3, you can scale down AND scale out!**

#### Option A: High-Throughput (shown in example)
| Pipeline Stage | Instances | Throughput | Cost/hr |
|----------------|-----------|------------|---------|
| BAM → BAMS3 | 1 × c5n.9xlarge | 365K reads/s | $0.97 |
| Variant Calling | 8 × c5n.18xlarge (EFA) | 2.9M reads/s | $15.36 |
| VCF Merge | 1 × c5n.9xlarge | 100 MB/s | $0.97 |

**Total:** Whole genome (3B reads) in **15-25 minutes** for **$5-8**

#### Option B: Cost-Optimized (small instances)
| Pipeline Stage | Instances | Throughput | Cost/hr |
|----------------|-----------|------------|---------|
| BAM → BAMS3 | 1 × c5.2xlarge | 90K reads/s | $0.34 |
| Variant Calling | 32 × c5.xlarge (spot) | 2.9M reads/s | $1.74 |
| VCF Merge | 1 × c5.xlarge | 50 MB/s | $0.17 |

**Total:** Whole genome (3B reads) in **20-30 minutes** for **$0.60-1.00** (85% cheaper!)

**Compare to traditional approach:**
- Sequential: 1 × r5.4xlarge (128GB RAM required) = 2-4 hours, $30-50
- BAMS3: 32 × c5.xlarge (8GB RAM each) = 25 minutes, $1 (30-50x cheaper!)

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ Stage 1: BAM → BAMS3 Conversion                              │
│ c5n.9xlarge (50 Gbps network)                                │
│                                                               │
│ S3 BAM (500GB) ──────┐                                       │
│                      ├→ Convert to chunks → S3 BAMS3         │
│                      │                                        │
│                      └→ Generate chunk manifest              │
└───────────────────────┬──────────────────────────────────────┘
                        │ S3 manifest
                        ▼
┌──────────────────────────────────────────────────────────────┐
│ Stage 2: Parallel Variant Calling (RDMA cluster)             │
│ 8 × c5n.18xlarge (100 Gbps EFA, same placement group)        │
│                                                               │
│ Instance 0 (Distributor):                                    │
│   ┌─────────────────┐                                        │
│   │ Read manifest   │                                        │
│   │ Download chunks │                                        │
│   │ ZMQ PUB (broadcast)──┐                                   │
│   └─────────────────┘    │                                   │
│                           ▼                                   │
│ Instances 1-7 (Workers):     (via RDMA: 100 Gbps)           │
│   ┌─────────────────┐                                        │
│   │ ZMQ SUB         │ ← Receive chunks                       │
│   │ Call variants   │                                        │
│   │ ZMQ PUSH        │ ──┐                                    │
│   └─────────────────┘   │                                    │
│                          │ VCF chunks via RDMA               │
└──────────────────────────┼──────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│ Stage 3: VCF Merge                                           │
│ c5n.9xlarge                                                  │
│                                                               │
│   ┌─────────────────┐                                        │
│   │ ZMQ PULL        │ ← Collect VCF chunks                   │
│   │ Merge in order  │                                        │
│   │ Index & upload  │ → S3 final VCF                         │
│   └─────────────────┘                                        │
└──────────────────────────────────────────────────────────────┘
```

## Why This Design?

### BAMS3: No More Giant Instances!

**The key insight:** With BAMS3, you don't need expensive high-memory instances because you never download the entire BAM file.

Traditional workflow requires giant instances:
```
Download 500GB BAM → Process → Upload results
Requires: r5.4xlarge (128GB RAM) = $1.01/hour
```

BAMS3 workflow uses commodity instances:
```
Stream only needed chunks → Process → Stream results
Uses: c5.xlarge (8GB RAM) = $0.17/hour (6x cheaper!)
```

**Why traditional workflows need big instances:**
- Must fit entire BAM in memory or cache
- samtools/bcftools load full index
- Random access = full file download with FUSE

**Why BAMS3 works with small instances:**
- Download only the 1-5MB chunks you're processing
- No indexes to load (metadata is tiny)
- Each chunk is independent - process and discard

### BAMS3: Cloud-Native Storage

Traditional BAM files are monolithic - you must download the entire file to access any region. BAMS3 splits the alignment into independent chunks:

**Traditional workflow:**
```bash
# Download entire 500GB BAM
aws s3 cp s3://bucket/sample.bam ./  # 10+ minutes
samtools view sample.bam chr1:1000000-2000000  # Query 1MB region
# Result: Downloaded 500GB to access 1MB
```

**BAMS3 workflow:**
```bash
# Download only needed chunks
bams3 query s3://bucket/sample.bams3 chr1:1000000-2000000
# Result: Downloaded 1-2MB, instant query
```

**For parallel processing:** Each worker can independently access its assigned chunks without coordination.

### ZeroMQ: Flexible Messaging Patterns

**Why not raw TCP?**
- Automatic reconnection on failures
- Built-in buffering (high water marks)
- Multiple patterns (PUB/SUB, PUSH/PULL, DEALER/ROUTER)
- Language-agnostic (Python, Go, C++, Rust)

**Pattern selection:**
- Stage 1 → 2: S3 (chunks pre-computed, not real-time)
- Within Stage 2: **PUB/SUB** (distributor broadcasts chunks to all workers)
- Stage 2 → 3: **PUSH/PULL** (workers push VCF chunks to merger)

### RDMA/EFA: Maximum Throughput

**Why RDMA for genomics?**

Variant calling on a 30x whole genome involves:
- **Input:** 500GB aligned BAM
- **Processing:** 3 billion reads, ~200 GB intermediate data
- **Output:** 5-10 GB VCF

With 8 workers processing in parallel, we need:
- **180 GB/s aggregate throughput** (each worker: 22.5 GB/s)
- Standard 10 Gbps network: 1.25 GB/s max (insufficient)
- EFA 100 Gbps network: 12.5 GB/s per instance (sufficient)

**RDMA benefits:**
- Zero-copy transfers (kernel bypass)
- Sub-microsecond latency
- Minimal CPU overhead (more for variant calling)
- Native MPI support for bioinformatics tools

## Setup

### Prerequisites

1. **AWS Account** with EFA-capable instance access
2. **BAMS3 tools** from [aws-direct-s3](https://github.com/scttfrdmn/aws-direct-s3)
3. **Genomics tools:** bcftools, samtools, tabix
4. **Python packages:** boto3, pysam, pyzmq

### Installation

```bash
# Install Spawn CLI
cd spawn
make install

# Install BAMS3 converter
git clone https://github.com/scttfrdmn/aws-direct-s3
cd aws-direct-s3/format-tools/bams3
pip install -r requirements.txt
sudo cp *.py /opt/bams3/

# Install genomics tools
sudo yum install -y bcftools samtools tabix

# Install ZeroMQ
pip install pyzmq
```

### AMI Preparation

Create a custom AMI with all tools pre-installed:

```bash
# Launch base instance
spawn launch --instance-type c5n.9xlarge --ami ami-xxxxxxxxx

# SSH and install tools
spawn ssh <instance-id>

# Follow installation steps above

# Create AMI
aws ec2 create-image \
  --instance-id i-xxxxxxxxx \
  --name genomics-pipeline-v1 \
  --description "BAMS3 + ZeroMQ + RDMA genomics tools"
```

## Usage

### 1. Validate Pipeline Definition

```bash
spawn pipeline validate examples/genomics/pipeline.json

# Output:
# ✓ Pipeline is valid
#
# Pipeline: BAMS3 Genomics Pipeline with RDMA
# ID: genomics-bams3-rdma
# Stages: 3
#
# Execution order:
#   1. bam-to-bams3
#   2. variant-calling
#   3. vcf-merge
#
# Features:
#   • Network streaming enabled
#   • EFA (Elastic Fabric Adapter) enabled
#   • Budget limit: $50.00
```

### 2. Configure Pipeline

Edit `pipeline.json` to set:

```json
{
  "stages": [
    {
      "stage_id": "bam-to-bams3",
      "env": {
        "INPUT_BAM": "s3://your-bucket/sample.bam",
        "OUTPUT_PREFIX": "s3://your-bucket/bams3/sample",
        "CHUNK_SIZE_MB": "5"
      }
    },
    {
      "stage_id": "variant-calling",
      "env": {
        "REFERENCE": "s3://broad-references/hg38/Homo_sapiens_assembly38.fasta"
      }
    }
  ],
  "s3_bucket": "your-pipeline-bucket",
  "notification_email": "you@example.com",
  "max_cost_usd": 50.0
}
```

### 3. Launch Pipeline

```bash
spawn pipeline launch examples/genomics/pipeline.json --wait

# Output:
# Launching pipeline: BAMS3 Genomics Pipeline with RDMA
# Pipeline ID: genomics-bams3-rdma
# Stages: 3
#
# Pipeline launch initiated.
# Waiting for completion...
#
# [Stage 1/3] bam-to-bams3: launching...
# [Stage 1/3] bam-to-bams3: running...
# [Stage 1/3] bam-to-bams3: completed (12m 34s)
#
# [Stage 2/3] variant-calling: launching...
# [Stage 2/3] variant-calling: running...
# [Stage 2/3] variant-calling: completed (18m 52s)
#
# [Stage 3/3] vcf-merge: launching...
# [Stage 3/3] vcf-merge: running...
# [Stage 3/3] vcf-merge: completed (3m 15s)
#
# Pipeline completed successfully!
# Total time: 34m 41s
# Total cost: $9.23
```

### 4. Monitor Progress

```bash
# Check status
spawn pipeline status genomics-bams3-rdma

# Watch in real-time
spawn pipeline status genomics-bams3-rdma --watch

# View stage logs
spawn ssh genomics-bams3-rdma --stage variant-calling --instance 2 \
  --command "tail -f /var/log/spawn/stage.log"
```

### 5. Collect Results

```bash
spawn pipeline collect genomics-bams3-rdma --output ./results/

# Downloads:
# ./results/
#   bams3/
#     HG00096/
#       _metadata.json
#       data/...
#   results/
#     HG00096_merged.vcf.gz
#     HG00096_merged.vcf.gz.tbi
#     merge_stats.json
```

## Performance Analysis

### Throughput Breakdown

**Stage 1: BAM → BAMS3 (1 instance)**
- Input: 500 GB BAM from S3
- Conversion: 365K reads/sec (12 minutes)
- Upload: 500 GB to S3 as chunks (8 minutes)
- **Total: 20 minutes, $0.32**

**Stage 2: Variant Calling (8 instances, RDMA)**
- Chunk distribution: 500 GB via RDMA (2 minutes at 100 Gbps)
- Parallel calling: 8 workers × 32 cores = 256 cores
- Throughput: 2.9M reads/sec aggregate
- VCF output streaming: 10 GB via RDMA (< 1 second)
- **Total: 12 minutes, $3.07**

**Stage 3: VCF Merge (1 instance)**
- Receive VCF chunks: 10 GB via RDMA (< 1 second)
- Merge and sort: 5 minutes
- Upload to S3: 2 GB compressed VCF (30 seconds)
- **Total: 6 minutes, $0.10**

**Pipeline total:** 38 minutes, $3.49

### Cost Optimization

**Baseline (no optimization):**
- Sequential processing: 1 × c5.9xlarge for 2 hours = $30.60
- Data transfer: Download 500GB + upload 2GB = $47.00
- **Total: $77.60**

**With BAMS3 + Spawn pipelines:**
- Parallel processing: 8 × c5n.18xlarge × 0.2hr = $3.07
- No data copying (chunks accessed directly): $0.00
- S3 GET requests: 100 chunks × $0.0004/1000 = negligible
- **Total: $3.49 (22x cheaper!)**

### Scaling

| Workers | Time | Cost | Reads/sec |
|---------|------|------|-----------|
| 1 | 96 min | $1.55 | 365K |
| 2 | 48 min | $1.55 | 730K |
| 4 | 24 min | $3.07 | 1.46M |
| 8 | 12 min | $3.07 | 2.92M |
| 16 | 6 min | $6.14 | 5.84M |

**Sweet spot:** 8 workers (linear scaling, minimal overhead)

## Troubleshooting

### Stage 1: Conversion Issues

**Problem:** Out of memory during conversion

**Solution:** Reduce chunk size or use larger instance

```json
{
  "env": {
    "CHUNK_SIZE_MB": "2"  // Smaller chunks (default: 5)
  }
}
```

### Stage 2: RDMA Not Working

**Problem:** Falling back to TCP, slow throughput

**Diagnosis:**
```bash
# Check EFA device
ssh instance-2
ls /sys/class/infiniband/

# Should show: efa_0

# Check libfabric
fi_info -p efa
```

**Solution:** Ensure:
- Instance type supports EFA (c5n, p4d, p5)
- Placement group configured
- `efa_enabled: true` in pipeline definition

### Stage 3: Merge Failures

**Problem:** VCF chunks out of order

**Solution:** Check chromosome naming consistency

```bash
# BAM uses "chr1", reference uses "1"
# Ensure consistent naming in conversion
```

## Advanced Usage

### Custom Variant Callers

Replace bcftools with GATK, DeepVariant, or others:

```python
# In variant_calling_rdma.py, replace:
subprocess.run([
    'bcftools', 'mpileup', ...
])

# With:
subprocess.run([
    'gatk', 'HaplotypeCaller',
    '-R', reference_local,
    '-I', chunk_bam,
    '-O', chunk_vcf,
    '-L', f"{chrom}:{start}-{end}"
])
```

### Multi-Sample Cohorts

Process 100 samples in parallel:

```bash
# Create pipeline for each sample
for sample in sample{1..100}.bam; do
  sed "s/SAMPLE_ID/$sample/g" pipeline.json > pipeline_${sample}.json
  spawn pipeline launch pipeline_${sample}.json --detached &
done
```

### Custom Chunk Sizes

Optimize chunk size for your use case:

- **Small regions (exomes):** 1 MB chunks
- **Whole genomes:** 5 MB chunks
- **High-coverage (>100x):** 10 MB chunks

## See Also

- [BAMS3 Specification](https://github.com/scttfrdmn/aws-direct-s3/blob/main/format-tools/bams3-spec.md)
- [Spawn Streaming Guide](../../docs/streaming.md)
- [AWS EFA Documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/efa.html)
- [ZeroMQ Guide](http://zguide.zeromq.org/)

## Citation

If you use this pipeline in your research, please cite:

```bibtex
@software{spawn_bams3_pipeline,
  author = {Freedman, Scott},
  title = {Spawn BAMS3 Genomics Pipeline},
  year = {2026},
  url = {https://github.com/scttfrdmn/mycelium/spawn/examples/genomics}
}
```
