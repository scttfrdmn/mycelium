"""
Truffle Python Bindings - Usage Examples
"""

from truffle import Truffle, ReservationType

# Initialize Truffle client
tf = Truffle()

# ============================================================================
# Example 1: Basic Instance Search
# ============================================================================

print("=" * 70)
print("Example 1: Search for m7i.large instances")
print("=" * 70)

results = tf.search("m7i.large", regions=["us-east-1", "us-west-2"])

for r in results:
    print(f"{r.instance_type} in {r.region}")
    print(f"  vCPUs: {r.vcpus}, Memory: {r.memory_gib} GiB")
    print(f"  Architecture: {r.architecture}")
    if r.availability_zones:
        print(f"  Available in {len(r.availability_zones)} AZs: {r.availability_zones}")
    print()

# ============================================================================
# Example 2: Find Graviton Instances with Filters
# ============================================================================

print("=" * 70)
print("Example 2: Find Graviton instances with 4+ vCPUs, 16+ GiB RAM")
print("=" * 70)

results = tf.search(
    "m8g.*",  # Graviton4 instances
    architecture="arm64",
    min_vcpus=4,
    min_memory=16,
    regions=["us-east-1"]
)

for r in results:
    print(f"{r.instance_type}: {r.vcpus} vCPUs, {r.memory_gib} GiB, {r.architecture}")

print()

# ============================================================================
# Example 3: Multi-AZ Deployment Planning
# ============================================================================

print("=" * 70)
print("Example 3: Find instances available in 3+ AZs")
print("=" * 70)

results = tf.az("m7i.xlarge", min_az_count=3, regions=["us-east-1"])

for r in results:
    az_count = len(r.availability_zones) if r.availability_zones else 0
    print(f"{r.instance_type} in {r.region}: {az_count} AZs")
    if r.availability_zones:
        print(f"  AZs: {', '.join(r.availability_zones)}")

print()

# ============================================================================
# Example 4: Find Cheap Spot Instances
# ============================================================================

print("=" * 70)
print("Example 4: Find cheapest Spot instances under $0.10/hr")
print("=" * 70)

spots = tf.spot(
    "m8g.*",
    max_price=0.10,
    sort_by_price=True,
    show_savings=True,
    regions=["us-east-1", "us-west-2"]
)

print(f"Found {len(spots)} Spot pricing options\n")

# Show top 5 cheapest
for spot in spots[:5]:
    print(f"{spot.instance_type} in {spot.availability_zone}: ${spot.spot_price:.4f}/hr", end="")
    if spot.savings_percent:
        print(f" (Save {spot.savings_percent:.1f}%!)")
    else:
        print()

print()

# ============================================================================
# Example 5: Check GPU Capacity for ML Training (Capacity Blocks)
# ============================================================================

print("=" * 70)
print("Example 5: Check Capacity Blocks for ML Training (p5.48xlarge)")
print("=" * 70)

try:
    blocks = tf.capacity(
        instance_types=["p5.48xlarge"],
        reservation_type=ReservationType.CAPACITY_BLOCKS,
        regions=["us-east-1"]
    )
    
    if blocks:
        print(f"Found {len(blocks)} Capacity Block(s) for ML training:\n")
        for block in blocks:
            print(f"Block ID: {block.capacity_block_id}")
            print(f"  Instance: {block.instance_count}x {block.instance_type}")
            print(f"  AZ: {block.availability_zone}")
            print(f"  Duration: {block.duration_hours} hours ({block.duration_hours // 24} days)")
            print(f"  Period: {block.start_date} to {block.end_date}")
            print(f"  State: {block.state}")
            print(f"  UltraCluster: {block.ultra_cluster_placement}")
            print()
    else:
        print("No Capacity Blocks found. You can create them via AWS Console/CLI.")
        print("Capacity Blocks are for reserved ML training (1-14 days, up to 8 weeks advance).")
    
except Exception as e:
    print(f"Note: {e}")

print()

# ============================================================================
# Example 6: Check GPU Capacity for ML Inference (ODCRs)
# ============================================================================

print("=" * 70)
print("Example 6: Check ODCRs for ML Inference (inf2, g6)")
print("=" * 70)

capacity = tf.capacity(
    instance_types=["inf2.xlarge", "g6.xlarge"],
    reservation_type=ReservationType.ODCR,
    available_only=True,
    regions=["us-east-1", "us-west-2"]
)

if capacity:
    print(f"Found {len(capacity)} ODCR(s) with available capacity:\n")
    for c in capacity:
        utilization = (c.used_capacity / c.total_capacity * 100) if c.total_capacity > 0 else 0
        print(f"{c.instance_type} in {c.availability_zone}")
        print(f"  Available: {c.available_capacity} instances")
        print(f"  Total: {c.total_capacity} instances")
        print(f"  Utilization: {utilization:.1f}%")
        print(f"  State: {c.state}")
        print()
else:
    print("No ODCRs with available capacity found.")
    print("ODCRs are for continuous workloads (ML inference, development).")

print()

# ============================================================================
# Example 7: Pre-Training Validation
# ============================================================================

print("=" * 70)
print("Example 7: Validate capacity before ML training run")
print("=" * 70)

def validate_training_capacity(instance_type: str, required_count: int, region: str) -> bool:
    """Validate we have enough capacity for training"""
    print(f"Validating: Need {required_count}x {instance_type} in {region}")
    
    # Check for Capacity Blocks
    blocks = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.CAPACITY_BLOCKS,
        regions=[region]
    )
    
    for block in blocks:
        if block.instance_count >= required_count and block.state in ["scheduled", "active"]:
            print(f"✅ Capacity Block found: {block.capacity_block_id}")
            print(f"   {block.instance_count}x {instance_type} reserved")
            return True
    
    # Check for ODCRs
    odcrs = tf.capacity(
        instance_types=[instance_type],
        reservation_type=ReservationType.ODCR,
        min_capacity=required_count,
        available_only=True,
        regions=[region]
    )
    
    if odcrs:
        odcr = odcrs[0]
        print(f"✅ ODCR found: {odcr.reservation_id}")
        print(f"   {odcr.available_capacity}x {instance_type} available")
        return True
    
    print(f"❌ No capacity found!")
    print(f"   Recommendation: Create ODCR or Capacity Block for {instance_type} in {region}")
    return False

# Validate before expensive training
has_capacity = validate_training_capacity("p5.48xlarge", 8, "us-east-1")

if has_capacity:
    print("\n✅ Ready to start training!")
else:
    print("\n⚠️  Cannot start training - create capacity reservation first")

print()

# ============================================================================
# Example 8: Cost Optimization - Find Best Price
# ============================================================================

print("=" * 70)
print("Example 8: Find best Spot price across regions")
print("=" * 70)

regions = ["us-east-1", "us-west-2", "eu-west-1"]
instance_type = "m7i.xlarge"

print(f"Comparing Spot prices for {instance_type} across {len(regions)} regions...\n")

all_spots = []
for region in regions:
    spots = tf.spot(instance_type, regions=[region], sort_by_price=True)
    if spots:
        all_spots.extend(spots)

# Sort all by price
all_spots.sort(key=lambda x: x.spot_price)

if all_spots:
    cheapest = all_spots[0]
    print(f"🏆 Best price: ${cheapest.spot_price:.4f}/hr")
    print(f"   Location: {cheapest.availability_zone}")
    print(f"   Instance: {cheapest.instance_type}")
    
    print(f"\nTop 3 cheapest locations:")
    for i, spot in enumerate(all_spots[:3], 1):
        print(f"{i}. {spot.availability_zone}: ${spot.spot_price:.4f}/hr")

print()

# ============================================================================
# Example 9: Monitor GPU Capacity (Polling)
# ============================================================================

print("=" * 70)
print("Example 9: Monitor GPU capacity (one-time check)")
print("=" * 70)

def check_gpu_capacity():
    """Check if any GPU capacity is available"""
    capacity = tf.capacity(
        gpu_only=True,
        available_only=True,
        min_capacity=1
    )
    
    if capacity:
        print(f"🎮 GPU Capacity Available!")
        for c in capacity:
            print(f"  {c.instance_type}: {c.available_capacity} instances in {c.availability_zone}")
        return True
    else:
        print("⏳ No GPU capacity currently available")
        return False

has_gpu = check_gpu_capacity()

print()

# ============================================================================
# Example 10: List All Regions
# ============================================================================

print("=" * 70)
print("Example 10: List all AWS regions")
print("=" * 70)

regions = tf.list_regions()
print(f"Available regions ({len(regions)}):")
for i in range(0, len(regions), 5):
    print("  " + ", ".join(regions[i:i+5]))

print("\n" + "=" * 70)
print("Examples complete!")
print("=" * 70)
