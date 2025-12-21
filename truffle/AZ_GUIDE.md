# Availability Zone (AZ) Guide

Truffle treats **Availability Zones (AZs) as first-class citizens** because AZ availability is often more critical than region availability for production deployments.

## Why AZs Matter More Than Regions

### Capacity Planning
- **Not all instance types are available in all AZs** within a region
- Some AZs may have capacity constraints during peak times
- Multi-AZ deployments require the instance type in 2+ AZs

### Real-World Scenarios

**❌ Region Check Alone is Insufficient:**
```bash
# This only tells you the region supports m7i.large
truffle search m7i.large --skip-azs
# Output: "m7i.large available in us-east-1" ✅
```

But when you try to launch:
- ❌ us-east-1a: Not available
- ❌ us-east-1b: Not available
- ✅ us-east-1c: Available
- ✅ us-east-1d: Available

**✅ AZ Check Reveals the Truth:**
```bash
# Default behavior - shows AZ details
truffle search m7i.large
# Output: "m7i.large in us-east-1: [us-east-1c, us-east-1d]" ✅
```

## Default Behavior: AZs Included

**Truffle includes AZ information by default** because it's critical for production:

```bash
# Default: Shows AZ details
truffle search m7i.large

# Faster but less detailed (region-level only)
truffle search m7i.large --skip-azs
```

## AZ-First Search Command

Use the dedicated `az` command for AZ-centric queries:

```bash
# Basic AZ search
truffle az m7i.large

# Filter by specific AZs
truffle az m7i.large --az us-east-1a,us-east-1b,us-east-1c

# Require minimum AZ count (for high availability)
truffle az "m8g.*" --min-az-count 3

# Find instances in specific AZs with specs
truffle az "*" --az us-west-2a,us-west-2b --min-vcpu 8
```

## Multi-AZ Deployment Validation

### Scenario: You need 3 AZs for high availability

```bash
# Find instance types available in at least 3 AZs per region
truffle az "m7i.*" --min-az-count 3

# More specific: exact AZs in us-east-1
truffle az m7i.xlarge --az us-east-1a,us-east-1b,us-east-1c
```

### Scenario: Active-Active across specific AZs

```bash
# Verify m7i.2xlarge in your target AZs
truffle az m7i.2xlarge \
  --az us-west-2a,us-west-2b,us-west-2c \
  --output json
```

## AZ Naming Conventions

### Format
- **Pattern:** `{region}{zone-letter}`
- **Example:** `us-east-1a`, `eu-west-1c`, `ap-southeast-2b`

### Important Notes

1. **AZ Names are Account-Specific**
   - `us-east-1a` in your account ≠ `us-east-1a` in another account
   - AWS randomizes AZ mappings for load distribution
   - Use AZ IDs for cross-account consistency (e.g., `use1-az1`)

2. **Not All Regions Have Same Number of AZs**
   - Most regions: 3-4 AZs
   - Some regions: 2 AZs
   - Newest regions: May have 3+ AZs

3. **Instance Type Availability Varies by AZ**
   - Newer instance types may not be in older AZs
   - Some instance types exclusive to specific AZs
   - Capacity can vary between AZs

## Common Use Cases

### 1. Multi-AZ Database Deployment

**Requirement:** PostgreSQL RDS across 3 AZs using r7i.xlarge

```bash
# Verify 3+ AZ availability
truffle az r7i.xlarge --min-az-count 3 --regions us-east-1

# Check specific AZs
truffle az r7i.xlarge \
  --az us-east-1a,us-east-1b,us-east-1c \
  --output json | jq -r '.[].availability_zones'
```

### 2. Kubernetes Multi-AZ Node Groups

**Requirement:** Worker nodes spread across AZs

```bash
# Find Graviton4 instances available in 3+ AZs
truffle az "m8g.*" \
  --min-az-count 3 \
  --min-vcpu 4 \
  --min-memory 16

# Specific region check
truffle az m8g.xlarge \
  --regions us-west-2 \
  --min-az-count 3
```

### 3. Disaster Recovery Planning

**Requirement:** Same instance type in DR region with 2+ AZs

```bash
# Primary region check
truffle az m7i.2xlarge --regions us-east-1 --min-az-count 2

# DR region check
truffle az m7i.2xlarge --regions us-west-2 --min-az-count 2

# Compare both
truffle search m7i.2xlarge --regions us-east-1,us-west-2
```

### 4. Cost Optimization with AZ Awareness

**Requirement:** Use Spot Instances, need AZ diversity

```bash
# Find instances with maximum AZ coverage for Spot diversity
truffle az "c7a.*" --min-az-count 3 --regions us-east-1

# Output sorted by AZ count (most first)
truffle az "c7a.xlarge" --output json | \
  jq 'sort_by(.availability_zones | length) | reverse'
```

### 5. Capacity Planning

**Requirement:** Identify AZs before launch

```bash
# Get exact AZ list for Terraform
truffle az m7i.large --regions us-east-1 --output json | \
  jq -r '.[0].availability_zones[]'

# Output: 
# us-east-1a
# us-east-1b
# us-east-1c
# us-east-1d
```

## Output Formats with AZ Data

### Table Format
```bash
truffle search m7i.large
```
```
+---------------+-----------+-------+--------------+--------------+------------------------+
| Instance Type | Region    | vCPUs | Memory (GiB) | Architecture | Availability Zones     |
+---------------+-----------+-------+--------------+--------------+------------------------+
| m7i.large     | us-east-1 |     2 |          8.0 | x86_64      | us-east-1a,us-east-1b, |
|               |           |       |              |              | us-east-1c,us-east-1d  |
+---------------+-----------+-------+--------------+--------------+------------------------+
```

### JSON Format
```bash
truffle az m7i.large --output json
```
```json
[
  {
    "instance_type": "m7i.large",
    "region": "us-east-1",
    "availability_zones": [
      "us-east-1a",
      "us-east-1b",
      "us-east-1c",
      "us-east-1d"
    ],
    "vcpus": 2,
    "memory_mib": 8192,
    "architecture": "x86_64"
  }
]
```

### CSV Format
```bash
truffle az m7i.large --output csv > az-data.csv
```
```csv
instance_type,region,vcpus,memory_gib,architecture,availability_zones
m7i.large,us-east-1,2,8.0,x86_64,us-east-1a;us-east-1b;us-east-1c;us-east-1d
```

## Integration Examples

### Terraform: Use Specific AZs
```bash
# Generate AZ list for Terraform
truffle az m7i.xlarge --regions us-west-2 --output json | \
  jq -r '.[0].availability_zones[]' | \
  awk '{printf "  \"%s\",\n", $0}'
```

Output for `terraform.tfvars`:
```hcl
availability_zones = [
  "us-west-2a",
  "us-west-2b",
  "us-west-2c",
]
```

### CloudFormation: Multi-AZ Subnets
```bash
# Verify AZ count before template deployment
AZ_COUNT=$(truffle az m7i.large --regions us-east-1 --output json | \
  jq '.[0].availability_zones | length')

if [ "$AZ_COUNT" -lt 3 ]; then
  echo "Error: Need at least 3 AZs"
  exit 1
fi
```

### Ansible: Dynamic Inventory
```bash
# Get AZs for dynamic instance placement
truffle az m8g.xlarge --regions eu-west-1 --output json | \
  jq -r '.[0].availability_zones[]' | \
  while read az; do
    echo "- name: instance_in_$az"
    echo "  az: $az"
  done
```

## Troubleshooting

### "Instance type not available in this AZ"

```bash
# Check which AZs DO support it
truffle az m7i.metal --regions us-east-1

# Try alternative instance types
truffle az "m7i.*" --min-az-count 3 --regions us-east-1
```

### "Insufficient capacity"

```bash
# Check all AZs - try others
truffle az m7i.xlarge --regions us-east-1

# Consider alternative generations
truffle az m6i.xlarge --regions us-east-1  # Previous gen
truffle az m8g.xlarge --regions us-east-1  # Graviton alternative
```

### Multi-AZ Deployment Failing

```bash
# Verify ALL required AZs support the instance
truffle az m7i.2xlarge \
  --az us-west-2a,us-west-2b,us-west-2c \
  --output json | \
  jq '.[] | select(.region == "us-west-2") | .availability_zones'
```

## Best Practices

### 1. Always Check AZs for Production
```bash
# ❌ Don't do this for production
truffle search m7i.large --skip-azs

# ✅ Do this
truffle search m7i.large  # AZs included by default
```

### 2. Require Minimum AZs for HA
```bash
# For high availability, require 3+ AZs
truffle az m7i.xlarge --min-az-count 3
```

### 3. Document AZ Selections
```bash
# Save AZ configuration
truffle az m7i.large --regions us-east-1 --output json > az-config.json
```

### 4. Test DR Parity
```bash
# Ensure DR region has same AZ coverage
PRIMARY_AZS=$(truffle az m7i.2xlarge --regions us-east-1 --output json | jq '.[0].availability_zones | length')
DR_AZS=$(truffle az m7i.2xlarge --regions us-west-2 --output json | jq '.[0].availability_zones | length')

if [ "$PRIMARY_AZS" != "$DR_AZs" ]; then
  echo "Warning: AZ count mismatch between regions"
fi
```

### 5. Use AZ Filters for Specific Deployments
```bash
# Only check your designated AZs
truffle az m7i.large --az us-east-1a,us-east-1c,us-east-1d
```

## Performance Considerations

### AZ Lookups are Slower
```bash
# Default (with AZs): ~30-60 seconds for all regions
truffle search m7i.large

# Fast mode (no AZs): ~10-20 seconds
truffle search m7i.large --skip-azs
```

**Recommendation:** Use full AZ lookup for production planning, use `--skip-azs` for quick queries.

### Optimize with Region Filters
```bash
# Faster: specific regions only
truffle az m7i.large --regions us-east-1,us-west-2

# Slower: all regions
truffle az m7i.large
```

## Summary

✅ **AZ data is included by default** - because it matters!  
✅ **Use `truffle az` command** for AZ-first perspective  
✅ **Use `--min-az-count`** for HA requirements  
✅ **Use `--az` filter** for specific AZ validation  
✅ **Use `--skip-azs`** only for quick, non-production queries  

**Remember:** Region availability ≠ AZ availability. Always verify at the AZ level for production deployments!
