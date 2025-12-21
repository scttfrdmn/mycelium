# Truffle Python Bindings (Native cgo)

**High-performance** Python wrapper for [Truffle](https://github.com/yourusername/truffle) using native Go library via cgo.

## 🚀 Performance

**Native cgo bindings** - not subprocess!

| Method | Call Overhead | Memory |
|--------|--------------|--------|
| **Native cgo** | **~1ms** | **Shared** |
| Subprocess | ~20-50ms | Isolated |

**10-50x faster** than subprocess-based wrappers!

## Installation

### 1. Install Go Compiler

```bash
# macOS
brew install go

# Ubuntu/Debian
sudo apt install golang-go

# Or download from https://golang.org/dl/
```

### 2. Install Python Package

```bash
pip install truffle-aws
```

The Go library is **automatically compiled** during pip install!

## Quick Start

```python
from truffle import Truffle

# Initialize client
tf = Truffle()

# Search for instance types
results = tf.search("m7i.large", regions=["us-east-1"])
for r in results:
    print(f"{r.instance_type}: {r.vcpus} vCPUs, {r.memory_gib} GiB")

# Get Spot pricing
spots = tf.spot("m8g.*", max_price=0.10, sort_by_price=True)
for s in spots[:5]:  # Top 5 cheapest
    print(f"{s.instance_type}: ${s.spot_price:.4f}/hr")

# Check GPU capacity for ML
from truffle import ReservationType

# For ML training (Capacity Blocks)
blocks = tf.capacity(
    instance_types=["p5.48xlarge"],
    reservation_type=ReservationType.CAPACITY_BLOCKS
)

# For ML inference (ODCRs)
capacity = tf.capacity(
    gpu_only=True,
    available_only=True,
    reservation_type=ReservationType.ODCR
)
```

## Features

- 🔍 **Instance Discovery**: Search by pattern, architecture, vCPUs, memory
- 📍 **AZ Information**: Find instances available in multiple AZs
- 💰 **Spot Pricing**: Real-time Spot prices across regions
- 🎮 **GPU Capacity**: Check ML capacity reservations (critical for p5, g6, trn1, inf2)
- 🐍 **Pythonic API**: Clean, type-hinted Python interface
- 📊 **Rich Types**: Dataclasses for all results

## ML Capacity Reservations

AWS offers **two types** of capacity reservations for ML workloads:

### 1. Capacity Blocks for ML (Training)

For **high-performance ML training**:
- **Duration**: 1-14 days
- **Booking**: Up to 8 weeks in advance
- **Instances**: P5 (H100), P4d (A100), Trn1 (Trainium)
- **Networking**: Co-located in EC2 UltraClusters
- **Use Case**: Large LLM training, distributed training

```python
from truffle import ReservationType

# Check training capacity blocks
blocks = tf.capacity(
    instance_types=["p5.48xlarge"],
    reservation_type=ReservationType.CAPACITY_BLOCKS,
    regions=["us-east-1"]
)

for block in blocks:
    print(f"{block.instance_count}x {block.instance_type}")
    print(f"  Duration: {block.duration_hours} hours")
    print(f"  Period: {block.start_date} to {block.end_date}")
    print(f"  UltraCluster: {block.ultra_cluster_placement}")
```

### 2. On-Demand Capacity Reservations / ODCRs (Inference/Continuous)

For **continuous or unpredictable workloads**:
- **Duration**: No fixed end date
- **Flexibility**: Create/cancel anytime
- **Use Case**: ML inference, development, high-availability services

```python
# Check inference capacity (ODCRs)
capacity = tf.capacity(
    instance_types=["inf2.xlarge", "g6.xlarge"],
    reservation_type=ReservationType.ODCR,
    available_only=True,
    min_capacity=10
)

for c in capacity:
    print(f"{c.instance_type}: {c.available_capacity} available")
    print(f"  Total: {c.total_capacity}, Used: {c.used_capacity}")
```

## API Reference

### Truffle Client

```python
class Truffle:
    def __init__(self, truffle_path: str = "truffle")
    
    def search(
        pattern: str,
        regions: Optional[List[str]] = None,
        architecture: Optional[str] = None,
        min_vcpus: Optional[int] = None,
        min_memory: Optional[float] = None,
        skip_azs: bool = False,
    ) -> List[InstanceType]
    
    def az(
        pattern: str,
        regions: Optional[List[str]] = None,
        min_az_count: Optional[int] = None,
        azs: Optional[List[str]] = None,
    ) -> List[InstanceType]
    
    def spot(
        pattern: str,
        regions: Optional[List[str]] = None,
        max_price: Optional[float] = None,
        show_savings: bool = False,
        sort_by_price: bool = False,
    ) -> List[SpotPrice]
    
    def capacity(
        instance_types: Optional[List[str]] = None,
        regions: Optional[List[str]] = None,
        reservation_type: ReservationType = ReservationType.ODCR,
        gpu_only: bool = False,
        available_only: bool = False,
        min_capacity: Optional[int] = None,
    ) -> List[CapacityReservation]
```

### Data Classes

```python
@dataclass
class InstanceType:
    instance_type: str
    region: str
    vcpus: int
    memory_gib: float
    architecture: str
    availability_zones: Optional[List[str]]

@dataclass
class SpotPrice:
    instance_type: str
    region: str
    availability_zone: str
    spot_price: float
    on_demand_price: Optional[float]
    savings_percent: Optional[float]

@dataclass
class CapacityReservation:
    reservation_id: str
    instance_type: str
    region: str
    availability_zone: str
    total_capacity: int
    available_capacity: int
    used_capacity: int
    state: str

@dataclass
class CapacityBlock:
    capacity_block_id: str
    instance_type: str
    instance_count: int
    availability_zone: str
    start_date: str
    end_date: str
    duration_hours: int
    state: str
    ultra_cluster_placement: bool
```

## Examples

### Instance Search

```python
# Find Graviton instances with filters
results = tf.search(
    "m8g.*",
    architecture="arm64",
    min_vcpus=4,
    min_memory=16,
    regions=["us-east-1", "us-west-2"]
)
```

### Multi-AZ Planning

```python
# Find instances in 3+ AZs (for HA)
results = tf.az("m7i.large", min_az_count=3)

for r in results:
    az_count = len(r.availability_zones)
    print(f"{r.instance_type}: {az_count} AZs")
```

### Spot Price Optimization

```python
# Find cheapest Spot instances
spots = tf.spot(
    "c8g.*",
    max_price=0.20,
    sort_by_price=True,
    show_savings=True
)

# Use cheapest
cheapest = spots[0]
print(f"Best: {cheapest.instance_type} @ ${cheapest.spot_price:.4f}/hr")
print(f"Save: {cheapest.savings_percent:.1f}% vs On-Demand")
```

### Pre-Training Validation

```python
def can_start_training(instance_type: str, count: int) -> bool:
    """Check if we have capacity for training"""
    
    # Check Capacity Blocks
    blocks = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.CAPACITY_BLOCKS
    )
    
    for block in blocks:
        if block.instance_count >= count and block.state == "active":
            print(f"✅ Capacity Block: {block.instance_count} instances")
            return True
    
    # Check ODCRs
    odcrs = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.ODCR,
        min_capacity=count,
        available_only=True
    )
    
    if odcrs:
        print(f"✅ ODCR: {odcrs[0].available_capacity} instances")
        return True
    
    print(f"❌ No capacity - create reservation first!")
    return False

# Before expensive training
if can_start_training("p5.48xlarge", 8):
    # Start training
    pass
```

### GPU Capacity Monitoring

```python
import time

def monitor_gpu_capacity(interval_seconds=300):
    """Monitor GPU capacity every 5 minutes"""
    while True:
        capacity = tf.capacity(
            gpu_only=True,
            available_only=True,
            min_capacity=1
        )
        
        if capacity:
            print(f"🎮 GPU Available:")
            for c in capacity:
                print(f"  {c.instance_type}: {c.available_capacity} in {c.availability_zone}")
            # Send alert, etc.
        
        time.sleep(interval_seconds)
```

### Integration with ML Frameworks

```python
# Ray cluster configuration
def get_optimal_gpu_az(instance_type: str, count: int) -> Optional[str]:
    """Find AZ with available GPU capacity"""
    capacity = tf.capacity(
        instance_types=[instance_type],
        available_only=True,
        min_capacity=count
    )
    
    if capacity:
        # Return AZ with most available capacity
        best = max(capacity, key=lambda c: c.available_capacity)
        return best.availability_zone
    return None

# Use for Ray cluster
gpu_az = get_optimal_gpu_az("p5.48xlarge", 8)
if gpu_az:
    # Launch Ray cluster in this AZ
    print(f"Launch in {gpu_az}")
```

## Convenience Functions

Quick access without creating Truffle object:

```python
from truffle import search_instances, get_spot_prices, check_gpu_capacity

# Quick search
instances = search_instances("m7i.*", min_vcpus=4)

# Quick Spot check
spots = get_spot_prices("m8g.large", max_price=0.10)

# Quick GPU capacity
gpu = check_gpu_capacity(available_only=True, min_capacity=1)
```

## Error Handling

```python
from truffle import TruffleError, TruffleNotFoundError

try:
    tf = Truffle()
    results = tf.search("m7i.large")
except TruffleNotFoundError:
    print("Truffle CLI not installed!")
    print("Install from: https://github.com/yourusername/truffle")
except TruffleError as e:
    print(f"Truffle error: {e}")
```

## Requirements

- Python 3.8+
- Go 1.22+ compiler (for building)
- AWS credentials configured (`aws login` or environment variables)

## Building from Source

```bash
cd bindings/python

# Build Go library and install
make install

# Or manually
make build  # Builds libtruffle.so/.dylib
pip install -e .
```

## Architecture

```
Python Code → ctypes/FFI → Go Shared Library (.so/.dylib) → AWS SDK
              (no subprocess overhead!)
```

**Benefits:**
- ✅ **Fast**: Direct function calls, no process spawning
- ✅ **Efficient**: Shared memory, no JSON serialization overhead  
- ✅ **Native**: True Go performance in Python
- ✅ **Simple**: Same API as subprocess version

## Authentication

Uses same AWS credentials as Truffle CLI:

```bash
# Easiest - AWS login
aws login

# Or environment variables
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...

# Or AWS profile
export AWS_PROFILE=my-profile
```

## Development

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run examples
python examples/usage.py

# Type checking
mypy truffle

# Formatting
black truffle
ruff check truffle
```

## License

MIT License - see [LICENSE](../../LICENSE) file

## Links

- **Main Project**: https://github.com/yourusername/truffle
- **Documentation**: https://github.com/yourusername/truffle/tree/main/bindings/python
- **Issues**: https://github.com/yourusername/truffle/issues

## Related

- [Truffle CLI Documentation](../../README.md)
- [Go Module Usage](../../MODULE_USAGE.md)
- [GPU Capacity Guide](../../GPU_CAPACITY_GUIDE.md)
- [Spot Instance Guide](../../SPOT_GUIDE.md)
