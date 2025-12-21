# ML Capacity Reservations: Complete Guide

AWS offers **TWO distinct types** of capacity reservations for machine learning workloads. Understanding the difference is critical for ML engineers!

## 🎯 The Two Types

### 1. **Capacity Blocks for ML** - For Training

**Designed for HIGH-PERFORMANCE ML TRAINING**

| Attribute | Details |
|-----------|---------|
| **Purpose** | Large-scale ML training & fine-tuning |
| **Duration** | 1-14 days (fixed) |
| **Booking** | Up to 8 weeks in advance |
| **Instances** | P5 (H100), P4d (A100), Trn1 (Trainium) |
| **Cluster Size** | 1-64 instances (up to 512 GPUs or 1024 Trainium chips) |
| **Networking** | Co-located in **EC2 UltraClusters** (low-latency, non-blocking) |
| **Pricing** | Reservation fee + Operating system fee |
| **Flexibility** | Fixed start/end dates, cannot be changed |
| **Best For** | Training GPT-scale models, distributed training runs |

**Check Capacity Blocks:**
```bash
truffle capacity --blocks
truffle capacity --blocks --instance-types p5.48xlarge
```

### 2. **On-Demand Capacity Reservations (ODCRs)** - For Inference/Continuous

**Designed for CONTINUOUS or UNPREDICTABLE WORKLOADS**

| Attribute | Details |
|-----------|---------|
| **Purpose** | ML inference, development, high-availability |
| **Duration** | No fixed end date (hold as long as needed) |
| **Booking** | Create/cancel immediately, anytime |
| **Instances** | Any EC2 instance type (P5, G6, Inf2, Trn1, general purpose) |
| **Cluster Size** | Any number of instances |
| **Networking** | Standard EC2 networking |
| **Pricing** | No reservation fee, only pay when instances are running |
| **Flexibility** | Create/cancel anytime, very flexible |
| **Best For** | Production inference APIs, ML dev environments, always-on services |

**Check ODCRs:**
```bash
truffle capacity  # Default
truffle capacity --odcr --gpu-only --available-only
```

## 📊 Quick Comparison Table

| Feature | Capacity Blocks for ML | ODCRs |
|---------|----------------------|-------|
| **Duration** | 1-14 days (fixed) | Indefinite (flexible) |
| **Advance Booking** | Up to 8 weeks | Immediate |
| **Cancellation** | Cannot cancel | Cancel anytime |
| **Networking** | UltraCluster (optimized) | Standard EC2 |
| **Commitment** | Full duration | No commitment |
| **Pricing Model** | Reservation + OS fee | Pay per use |
| **Target Workload** | ML Training | ML Inference/Dev |
| **Typical Cost** | Higher (reserved) | Lower (on-demand) |

## 🎓 When to Use Which?

### Use **Capacity Blocks for ML** when:

✅ **Training large language models** (GPT-4 scale, 100B+ parameters)  
✅ **Distributed training** across multiple nodes  
✅ **Fixed training schedules** (know exact start/end times)  
✅ **Need guaranteed performance** (UltraCluster networking)  
✅ **Multi-week planning** (can book 8 weeks ahead)  
✅ **Budget is approved** for specific training runs  

**Example Use Cases:**
- Pre-training a new LLM foundation model
- Large-scale fine-tuning run
- Multi-node reinforcement learning
- Research project with fixed timeline

### Use **ODCRs** when:

✅ **Running ML inference APIs** (24/7 prediction serving)  
✅ **Development and experimentation** (need flexibility)  
✅ **High-availability services** (no scheduled downtime)  
✅ **Unpredictable workloads** (don't know exact timing)  
✅ **Want flexibility** to start/stop anytime  
✅ **Building production services** that need guaranteed capacity  

**Example Use Cases:**
- Production inference endpoint for customer-facing app
- ML development environment
- Continuous model training pipeline
- Auto-scaling inference clusters

## 🚀 Using Truffle to Check Both Types

### Check Capacity Blocks (Training)

```bash
# All Capacity Blocks
truffle capacity --blocks

# Specific GPU instances for training
truffle capacity --blocks --instance-types p5.48xlarge,p4d.24xlarge

# Only active/scheduled blocks
truffle capacity --blocks --active-only

# In specific regions
truffle capacity --blocks --instance-types p5.48xlarge --regions us-east-1,us-west-2
```

**Output:**
```
📊 Capacity Block Summary:
   Total Blocks: 3
   Instance Types: 2 (p5.48xlarge, trn1.32xlarge)
   Duration Range: 168-336 hours (7-14 days)

+---------------+-------+-------------+------------+------------+----------+----------+
| Instance Type | Count | AZ          | Start      | End        | Duration | State    |
+---------------+-------+-------------+------------+------------+----------+----------+
| p5.48xlarge   | 32    | us-east-1a | 2025-01-15 | 2025-01-22 | 168h     | scheduled|
| p5.48xlarge   | 16    | us-west-2b | 2025-01-20 | 2025-02-03 | 336h     | scheduled|
| trn1.32xlarge | 64    | us-east-1c | 2025-01-18 | 2025-01-25 | 168h     | scheduled|
+---------------+-------+-------------+------------+------------+----------+----------+

UltraCluster Placement: Yes (optimized networking for distributed training)
```

### Check ODCRs (Inference/Continuous)

```bash
# All ODCRs
truffle capacity

# Only GPU ODCRs with available capacity
truffle capacity --odcr --gpu-only --available-only

# Specific instances for inference
truffle capacity --odcr --instance-types inf2.xlarge,g6.xlarge

# Minimum capacity needed
truffle capacity --odcr --gpu-only --min-capacity 10
```

**Output:**
```
📊 Capacity Reservation Summary (ODCRs):
   Total Reservations: 8
   Instance Types: 4 (inf2.xlarge, g6.xlarge, g6.2xlarge, p4d.24xlarge)
   Total Capacity: 128 instances
   Available: 42 instances (32.8% free)

+---------------+-----------+-------------+-------+-----------+------+--------+
| Instance Type | Region    | AZ          | Total | Available | Used | State  |
+---------------+-----------+-------------+-------+-----------+------+--------+
| inf2.xlarge   | us-east-1 | us-east-1a | 32    | 12        | 20   | active |
| g6.xlarge     | us-west-2 | us-west-2a | 64    | 24        | 40   | active |
| g6.2xlarge    | eu-west-1 | eu-west-1b | 16    | 4         | 12   | active |
| p4d.24xlarge  | us-east-1 | us-east-1c | 16    | 2         | 14   | active |
+---------------+-----------+-------------+-------+-----------+------+--------+

No fixed end date - can use indefinitely
```

## 💡 Real-World Scenarios

### Scenario 1: Train GPT-4 Scale Model

**Requirement:** Train 175B parameter model over 10 days

**Solution: Capacity Blocks for ML** ✅

```bash
# Check available Capacity Blocks
truffle capacity --blocks --instance-types p5.48xlarge

# Book 64x p5.48xlarge (512 H100 GPUs)
# - Duration: 240 hours (10 days)
# - Book 4 weeks in advance
# - UltraCluster placement for optimal networking
```

**Why Capacity Blocks:**
- Need 512 GPUs for 10 days straight
- Know exact training schedule
- Require UltraCluster networking for multi-node training
- Can plan and book 4 weeks ahead

**Cost:** Reservation fee + 10 days of usage

### Scenario 2: Production ML Inference API

**Requirement:** 24/7 inference API serving 1M requests/day

**Solution: ODCRs** ✅

```bash
# Reserve inference capacity
truffle capacity --odcr --instance-types inf2.xlarge --min-capacity 20

# Auto-scaling: 20-100 instances based on demand
# ODCR guarantees base 20 can always launch
```

**Why ODCRs:**
- Continuous service (no fixed end date)
- Need flexibility to scale up/down
- Don't know exact usage patterns
- Want guaranteed capacity for base load

**Cost:** Pay only for instances actually running

### Scenario 3: ML Research & Experimentation

**Requirement:** Research team needs GPU access for various experiments

**Solution: ODCRs** ✅

```bash
# Reserve development capacity
truffle capacity --odcr --instance-types g6.xlarge --available-only

# Team can launch instances anytime
# No fixed schedule
```

**Why ODCRs:**
- Unpredictable workload
- Need flexibility
- Various experiment durations
- Want guaranteed access when needed

### Scenario 4: Quarterly Model Retraining

**Requirement:** Retrain model every quarter, 7 days each time

**Solution: Capacity Blocks for ML** ✅

```bash
# Book next quarter's training block
# 8 weeks in advance, each quarter

# Q1 2025: Book in November 2024
truffle capacity --blocks --instance-types p5.48xlarge

# Reserve for specific dates
# - Q1: Jan 15-22 (7 days)
# - Q2: Apr 15-22 (7 days)
# - Q3: Jul 15-22 (7 days)
# - Q4: Oct 15-22 (7 days)
```

**Why Capacity Blocks:**
- Predictable schedule
- Fixed duration training runs
- Can book well in advance
- Need guaranteed capacity on specific dates

## 🔧 Python Bindings Examples

### Check Capacity Blocks

```python
from truffle import Truffle, ReservationType

tf = Truffle()

# Check training capacity blocks
blocks = tf.capacity(
    instance_types=["p5.48xlarge"],
    reservation_type=ReservationType.CAPACITY_BLOCKS,
    regions=["us-east-1"]
)

for block in blocks:
    print(f"Capacity Block: {block.capacity_block_id}")
    print(f"  Instances: {block.instance_count}x {block.instance_type}")
    print(f"  Duration: {block.duration_hours} hours ({block.duration_hours // 24} days)")
    print(f"  Period: {block.start_date} to {block.end_date}")
    print(f"  UltraCluster: {block.ultra_cluster_placement}")
```

### Check ODCRs

```python
# Check inference capacity (ODCRs)
capacity = tf.capacity(
    instance_types=["inf2.xlarge", "g6.xlarge"],
    reservation_type=ReservationType.ODCR,
    available_only=True,
    min_capacity=10
)

for c in capacity:
    print(f"ODCR: {c.reservation_id}")
    print(f"  Instance: {c.instance_type}")
    print(f"  Available: {c.available_capacity}/{c.total_capacity}")
    print(f"  Location: {c.availability_zone}")
```

### Pre-Training Validation

```python
def validate_training_capacity(instance_type: str, count: int, duration_days: int):
    """Check if we have capacity for training"""
    
    # Check Capacity Blocks (preferred for training)
    blocks = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.CAPACITY_BLOCKS
    )
    
    for block in blocks:
        block_days = block.duration_hours // 24
        if (block.instance_count >= count and 
            block_days >= duration_days and
            block.state in ["scheduled", "active"]):
            print(f"✅ Capacity Block available:")
            print(f"   {block.instance_count} instances for {block_days} days")
            print(f"   UltraCluster networking enabled")
            return True
    
    # Fallback to ODCRs
    odcrs = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.ODCR,
        min_capacity=count,
        available_only=True
    )
    
    if odcrs:
        print(f"⚠️  No Capacity Block found")
        print(f"✅ ODCR available: {odcrs[0].available_capacity} instances")
        print(f"   Note: Standard networking (not UltraCluster)")
        return True
    
    print(f"❌ No capacity available!")
    print(f"   Create Capacity Block or ODCR for {instance_type}")
    return False

# Before training
if validate_training_capacity("p5.48xlarge", 32, 7):
    print("Ready to train!")
```

## 🛠️ Management Tools (2025)

### New AWS Features for ML Capacity

**1. Capacity Reservation Topology API** (Late 2025)
- View hierarchical location of reservations
- Optimize job scheduling
- Better node ranking for distributed ML

**2. AWS Cost and Usage Reports (CUR 2.0)**
- Hourly, resource-level granularity
- Track utilization of ML reservations
- Better cost visibility

**3. Service Integrations**
- **Amazon SageMaker**: Direct integration with Capacity Blocks
- **Amazon EKS**: Use ODCRs for GPU node groups
- **AWS ParallelCluster (PCS)**: Cluster-aware capacity management

**Check with Truffle:**
```bash
# Export for Terraform/analysis
truffle capacity --blocks --output json > capacity-blocks.json
truffle capacity --odcr --output json > odcrs.json
```

## ⚠️ Common Mistakes

### ❌ Wrong: Using ODCRs for Fixed Training Runs

```bash
# Don't do this for 7-day training run
truffle capacity --odcr --instance-types p5.48xlarge
```

**Problem:**
- No UltraCluster networking
- Not optimized for multi-node training
- Miss out on training-specific features

**✅ Right: Use Capacity Blocks**

```bash
truffle capacity --blocks --instance-types p5.48xlarge
```

### ❌ Wrong: Using Capacity Blocks for Inference

```bash
# Don't do this for 24/7 inference API
truffle capacity --blocks --instance-types inf2.xlarge
```

**Problem:**
- Fixed 1-14 day duration (API needs to run indefinitely)
- Expensive for continuous workload
- Wrong pricing model

**✅ Right: Use ODCRs**

```bash
truffle capacity --odcr --instance-types inf2.xlarge
```

## 📋 Decision Tree

```
Need GPU capacity for ML?
│
├─ For TRAINING?
│  │
│  ├─ Fixed schedule (1-14 days)?
│  │  └─ ✅ Use Capacity Blocks for ML
│  │     - Book up to 8 weeks ahead
│  │     - UltraCluster networking
│  │     - Optimal for distributed training
│  │
│  └─ Continuous/unpredictable?
│     └─ ✅ Use ODCRs
│        - Flexible duration
│        - Create/cancel anytime
│
└─ For INFERENCE/DEV?
   └─ ✅ Use ODCRs
      - 24/7 availability
      - No fixed duration
      - Perfect for production APIs
```

## 🎯 Summary

| **Use Case** | **Reservation Type** | **Why** |
|--------------|---------------------|---------|
| Training GPT-scale models | Capacity Blocks | UltraCluster, fixed duration, advance booking |
| Distributed training (multi-node) | Capacity Blocks | Optimized networking, co-located GPUs |
| Production inference API | ODCRs | Continuous availability, flexible |
| ML development environment | ODCRs | Flexibility, no commitment |
| Quarterly model retraining | Capacity Blocks | Predictable schedule, advance booking |
| Auto-scaling inference | ODCRs | Variable capacity needs |

**Key Insight:**  
**Capacity Blocks** = Training (short, intense, scheduled)  
**ODCRs** = Inference/Continuous (long-running, flexible)

---

**Check your capacity:**

```bash
# Training capacity
truffle capacity --blocks --gpu-only

# Inference capacity
truffle capacity --odcr --gpu-only --available-only
```

See [Python Bindings](../bindings/python/README.md) for programmatic access!
