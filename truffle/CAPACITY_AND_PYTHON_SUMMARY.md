# ✅ ML Capacity Types + Python Bindings - Complete!

You asked for two critical clarifications/features and they're both done! 🎉

## 1. ✅ Clarified: TWO Types of ML Capacity Reservations

### The Distinction

AWS offers **TWO completely different** capacity reservation types for ML:

#### **Type 1: Capacity Blocks for ML** 🎯 TRAINING

**For HIGH-PERFORMANCE ML TRAINING**

| Feature | Details |
|---------|---------|
| Purpose | Large-scale training & fine-tuning |
| Duration | **1-14 days** (fixed) |
| Booking | **Up to 8 weeks in advance** |
| Instances | P5 (H100), P4d (A100), Trn1 (Trainium) |
| Networking | **EC2 UltraClusters** (low-latency) |
| Cluster | 1-64 instances (512 GPUs, 1024 Trainium) |
| Pricing | Reservation fee + OS fee |

**Use for:** Training GPT-4 scale models, distributed training

**Check:**
```bash
truffle capacity --blocks --instance-types p5.48xlarge
```

#### **Type 2: On-Demand Capacity Reservations (ODCRs)** 🎯 INFERENCE

**For CONTINUOUS/UNPREDICTABLE WORKLOADS**

| Feature | Details |
|---------|---------|
| Purpose | ML inference, development, HA services |
| Duration | **No fixed end date** (indefinite) |
| Booking | **Immediate** (create/cancel anytime) |
| Instances | Any EC2 instance type |
| Networking | Standard EC2 |
| Cluster | Any number |
| Pricing | Pay only when running |

**Use for:** Production inference APIs, ML development

**Check:**
```bash
truffle capacity --odcr --gpu-only --available-only
```

### Quick Comparison

| | Capacity Blocks | ODCRs |
|---|---|---|
| **Training** | ✅ Perfect | ⚠️ Not optimized |
| **Inference** | ❌ Wrong tool | ✅ Perfect |
| **Duration** | 1-14 days | Indefinite |
| **Networking** | UltraCluster | Standard |
| **Flexibility** | Fixed | Very flexible |

---

## 2. ✅ Python Bindings

Complete Python wrapper with full ML capacity support!

### Installation

```bash
# 1. Install Truffle CLI
curl -LO https://github.com/yourusername/truffle/releases/latest/download/truffle-linux-amd64
chmod +x truffle-linux-amd64
sudo mv truffle-linux-amd64 /usr/local/bin/truffle

# 2. Install Python package
pip install truffle-aws
```

### Basic Usage

```python
from truffle import Truffle, ReservationType

tf = Truffle()

# Search instances
results = tf.search("m7i.large", regions=["us-east-1"])

# Get Spot pricing
spots = tf.spot("m8g.*", max_price=0.10, sort_by_price=True)

# Check Capacity Blocks (for TRAINING)
blocks = tf.capacity(
    instance_types=["p5.48xlarge"],
    reservation_type=ReservationType.CAPACITY_BLOCKS
)

# Check ODCRs (for INFERENCE)
capacity = tf.capacity(
    gpu_only=True,
    available_only=True,
    reservation_type=ReservationType.ODCR
)
```

### Full API

```python
class Truffle:
    # Instance search
    def search(pattern, regions, architecture, min_vcpus, min_memory)
    
    # AZ-centric search
    def az(pattern, regions, min_az_count, azs)
    
    # Spot pricing
    def spot(pattern, regions, max_price, show_savings, sort_by_price)
    
    # ML capacity (both types!)
    def capacity(
        instance_types,
        regions,
        reservation_type,  # CAPACITY_BLOCKS or ODCR
        gpu_only,
        available_only,
        min_capacity
    )
```

---

## 🎯 Real-World Examples

### Example 1: Training LLM (Capacity Blocks)

**Scenario:** Train 175B parameter model, 10 days

**Python:**
```python
# Check training capacity
blocks = tf.capacity(
    instance_types=["p5.48xlarge"],
    reservation_type=ReservationType.CAPACITY_BLOCKS,
    regions=["us-east-1"]
)

for block in blocks:
    if block.instance_count >= 32 and block.duration_hours >= 240:
        print(f"✅ Capacity Block: {block.capacity_block_id}")
        print(f"   {block.instance_count}x H100 GPUs")
        print(f"   UltraCluster: {block.ultra_cluster_placement}")
        print(f"   Period: {block.start_date} to {block.end_date}")
        # Book this block!
```

**CLI:**
```bash
truffle capacity --blocks --instance-types p5.48xlarge --regions us-east-1
```

**Why Capacity Blocks:**
- 10-day fixed duration ✅
- Need 256 H100s (32x p5.48xlarge) ✅
- Distributed training = need UltraCluster ✅
- Can book 4 weeks ahead ✅

### Example 2: Inference API (ODCRs)

**Scenario:** 24/7 ML inference, auto-scale 10-100 instances

**Python:**
```python
# Check inference capacity
capacity = tf.capacity(
    instance_types=["inf2.xlarge"],
    reservation_type=ReservationType.ODCR,
    available_only=True,
    min_capacity=100  # Need 100 for peak load
)

if capacity:
    c = capacity[0]
    print(f"✅ ODCR: {c.reservation_id}")
    print(f"   Available: {c.available_capacity} instances")
    print(f"   Location: {c.availability_zone}")
    # Deploy inference API
```

**CLI:**
```bash
truffle capacity --odcr --instance-types inf2.xlarge --min-capacity 100
```

**Why ODCRs:**
- 24/7 service (no fixed end date) ✅
- Variable load (10-100 instances) ✅
- Need flexibility ✅
- Standard networking sufficient ✅

### Example 3: Pre-Training Validation

**Python:**
```python
def can_train(instance_type, count, duration_days):
    """Validate capacity before expensive training"""
    
    # Check Capacity Blocks first (preferred for training)
    blocks = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.CAPACITY_BLOCKS
    )
    
    for block in blocks:
        if (block.instance_count >= count and
            block.duration_hours >= duration_days * 24 and
            block.state in ["scheduled", "active"]):
            print(f"✅ Capacity Block found!")
            print(f"   {block.instance_count} instances")
            print(f"   UltraCluster networking")
            return True
    
    # Fallback to ODCRs (not ideal but works)
    odcrs = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.ODCR,
        min_capacity=count,
        available_only=True
    )
    
    if odcrs:
        print(f"⚠️  ODCR found (no Capacity Block)")
        print(f"   {odcrs[0].available_capacity} instances")
        print(f"   Standard networking (not UltraCluster)")
        return True
    
    print(f"❌ No capacity!")
    print(f"   Create Capacity Block for {instance_type}")
    return False

# Before $100k training run
if can_train("p5.48xlarge", 32, 7):
    start_training()
else:
    create_capacity_block()
```

---

## 📦 What's New

### Updated Files

1. **cmd/capacity.go** - Added `--blocks` and `--odcr` flags
2. **pkg/aws/client.go** - Added Capacity Block types and methods
3. **GPU_CAPACITY_GUIDE.md** - Updated with both types
4. **ML_CAPACITY_TYPES.md** - NEW! Complete guide to both types

### New: Python Bindings

5. **bindings/python/truffle/__init__.py** - Full Python API
6. **bindings/python/setup.py** - Package setup
7. **bindings/python/README.md** - Python documentation
8. **bindings/python/examples/usage.py** - 10 complete examples

### Documentation

9. Updated **README.md** with capacity types clarification
10. Updated all capacity examples to distinguish types

---

## 🎓 Key Learnings

### Wrong Assumptions (Before)

❌ "All capacity reservations are the same"  
❌ "Just check for 'capacity reservations'"  
❌ "Region availability = instance availability"

### Correct Understanding (Now)

✅ **Two distinct types:** Capacity Blocks (training) vs ODCRs (inference)  
✅ **Different purposes:** Training needs UltraCluster, inference needs flexibility  
✅ **Check the right type:** Use `--blocks` for training, `--odcr` for inference  
✅ **AZ-level granularity:** Both types are AZ-specific  

---

## 🚀 Command Reference

### Capacity Blocks (Training)

```bash
# All Capacity Blocks
truffle capacity --blocks

# Specific training instances
truffle capacity --blocks --instance-types p5.48xlarge,trn1.32xlarge

# Active/scheduled blocks only
truffle capacity --blocks --active-only

# In specific regions
truffle capacity --blocks --instance-types p5.48xlarge --regions us-east-1
```

### ODCRs (Inference/Continuous)

```bash
# All ODCRs (default)
truffle capacity

# GPU ODCRs with capacity
truffle capacity --odcr --gpu-only --available-only

# Inference instances
truffle capacity --odcr --instance-types inf2.xlarge,g6.xlarge

# Minimum capacity needed
truffle capacity --odcr --gpu-only --min-capacity 10
```

### Python

```python
# Training capacity
blocks = tf.capacity(
    instance_types=["p5.48xlarge"],
    reservation_type=ReservationType.CAPACITY_BLOCKS
)

# Inference capacity
capacity = tf.capacity(
    instance_types=["inf2.xlarge"],
    reservation_type=ReservationType.ODCR,
    available_only=True
)
```

---

## 📊 Use Case Matrix

| Use Case | Type | Command |
|----------|------|---------|
| **Train GPT-4 scale model** | Capacity Blocks | `--blocks --instance-types p5.48xlarge` |
| **Distributed training** | Capacity Blocks | `--blocks --instance-types trn1.32xlarge` |
| **Production inference API** | ODCR | `--odcr --instance-types inf2.xlarge` |
| **ML development** | ODCR | `--odcr --gpu-only` |
| **Auto-scaling inference** | ODCR | `--odcr --available-only --min-capacity 100` |
| **Quarterly retraining** | Capacity Blocks | `--blocks` (book quarterly) |
| **Research experiments** | ODCR | `--odcr --gpu-only` |

---

## 🎯 Decision Tree

```
Do you need GPU capacity?
│
├─ For TRAINING?
│  │
│  ├─ Fixed 1-14 day schedule?
│  │  └─ ✅ Capacity Blocks for ML
│  │     Python: reservation_type=CAPACITY_BLOCKS
│  │     CLI: truffle capacity --blocks
│  │
│  └─ Continuous/unpredictable?
│     └─ ✅ ODCRs
│        Python: reservation_type=ODCR
│        CLI: truffle capacity --odcr
│
└─ For INFERENCE/DEV?
   └─ ✅ ODCRs
      Python: reservation_type=ODCR
      CLI: truffle capacity --odcr
```

---

## 📚 Documentation

**Comprehensive Guides:**
- **ML_CAPACITY_TYPES.md** - Complete guide to both types
- **GPU_CAPACITY_GUIDE.md** - GPU/ML capacity strategies
- **bindings/python/README.md** - Python API documentation

**Examples:**
- **bindings/python/examples/usage.py** - 10 Python examples
- All CLI examples updated with `--blocks` and `--odcr`

---

## ✅ Summary

**Both requests completed:**

1. ✅ **ML Capacity Types Clarified**
   - Two distinct types: Capacity Blocks vs ODCRs
   - Clear documentation on when to use each
   - Updated all commands and examples
   - Added `--blocks` and `--odcr` flags

2. ✅ **Python Bindings**
   - Full Python API
   - Support for both capacity types
   - Type-hinted, dataclasses
   - 10 working examples
   - Clean Pythonic interface

**You can now:**
- Distinguish between training and inference capacity
- Check Capacity Blocks for ML training runs
- Check ODCRs for continuous workloads
- Use Truffle from Python applications
- Build automated capacity monitoring
- Integrate with ML frameworks

---

**Get started:**

```bash
# CLI - Training capacity
truffle capacity --blocks --instance-types p5.48xlarge

# CLI - Inference capacity
truffle capacity --odcr --gpu-only --available-only

# Python
pip install truffle-aws
```

```python
from truffle import Truffle, ReservationType

tf = Truffle()

# Training
blocks = tf.capacity(
    instance_types=["p5.48xlarge"],
    reservation_type=ReservationType.CAPACITY_BLOCKS
)

# Inference
capacity = tf.capacity(
    gpu_only=True,
    reservation_type=ReservationType.ODCR,
    available_only=True
)
```

See **ML_CAPACITY_TYPES.md** and **bindings/python/README.md** for complete documentation! 🚀
