# Spot Instance Guide

Truffle is **highly useful for Spot instances** - here's everything you need to know!

## 🎯 Why Truffle is Perfect for Spot

### 1. **AZ Diversity is Critical for Spot** ✨

Spot instances work best when you can:
- Launch across **multiple AZs** (better availability)
- Switch between AZs when capacity is low
- Find alternative AZs with better pricing

**Truffle shows you exactly which AZs have your instance type** - this is crucial for Spot diversity!

### 2. **Spot Pricing Varies by AZ**

Different availability zones have different Spot prices for the **same instance type**:
- us-east-1a: m7i.large @ $0.0312/hr
- us-east-1b: m7i.large @ $0.0405/hr  
- us-east-1c: m7i.large @ $0.0298/hr ← Cheapest!

**Truffle's new `spot` command** shows you these prices!

## 🚀 Quick Start

### Find Spot Instances

```bash
# Get current Spot prices
truffle spot m7i.large

# Find cheap Spot instances
truffle spot "m8g.*" --max-price 0.10

# Sort by price (cheapest first)
truffle spot "c7i.*" --sort-by-price

# Show savings vs On-Demand
truffle spot m7i.xlarge --show-savings
```

### Check AZ Availability (Current Feature)

```bash
# See which AZs have the instance (for Spot diversity)
truffle search m7i.large

# Or use AZ command
truffle az m7i.large --min-az-count 3
```

## 💰 Spot Command (NEW!)

The `truffle spot` command provides real-time Spot pricing:

### Basic Usage

```bash
truffle spot m7i.large
```

Output:
```
💰 Spot Instance Summary:
   Instance Types: 1
   Regions: 17
   Availability Zones: 52
   Price Range: $0.0298 - $0.0512 per hour
   Average Price: $0.0387 per hour

+---------------+-----------+------------------+----------------+
| Instance Type | Region    | Availability Zone| Spot Price/hr  |
+---------------+-----------+------------------+----------------+
| m7i.large     | us-east-1 | us-east-1a      | $0.0312        |
|               |           | us-east-1b      | $0.0405        |
|               |           | us-east-1c      | $0.0298        |← Cheapest!
|               |           | us-east-1d      | $0.0354        |
|               | us-west-2 | us-west-2a      | $0.0342        |
|               |           | us-west-2b      | $0.0389        |
+---------------+-----------+------------------+----------------+
```

### Find Cheap Spot Instances

```bash
# Find instances under $0.10/hour
truffle spot "m7i.*" --max-price 0.10

# Sort by price
truffle spot "c8g.*" --sort-by-price
```

### Compare Savings

```bash
# Show savings vs On-Demand
truffle spot r7i.xlarge --show-savings
```

Output shows:
- Spot price
- On-Demand price
- Savings percentage (typically 50-90%!)

## 📊 Common Spot Use Cases

### 1. Multi-AZ Spot Fleet

**Goal:** Launch Spot instances across 3+ AZs for redundancy

```bash
# Find instance available in 3+ AZs
truffle az m7i.large --min-az-count 3

# Get pricing for those AZs
truffle spot m7i.large --regions us-east-1
```

**Why this works:**
- More AZs = better Spot availability
- Can failover to another AZ if interrupted
- Diversify across price zones

### 2. Cost Optimization

**Goal:** Find cheapest Spot instances for your workload

```bash
# Find all Graviton instances under $0.15/hr
truffle spot "m8g.*" --max-price 0.15 --sort-by-price

# Compare to Intel
truffle spot "m7i.*" --max-price 0.15 --sort-by-price

# Show actual savings
truffle spot m8g.xlarge --show-savings
```

### 3. Kubernetes Spot Node Groups

**Goal:** Configure multiple AZ node groups for EKS

```bash
# Find Graviton instances in 3+ AZs
truffle az "m8g.xlarge" --min-az-count 3 --regions us-west-2

# Get Spot pricing for capacity planning
truffle spot "m8g.xlarge" --regions us-west-2

# Export for Terraform
truffle spot "m8g.xlarge" --regions us-west-2 --output json > spot-config.json
```

### 4. Batch Processing

**Goal:** Run batch jobs on cheapest Spot instances

```bash
# Find compute-optimized instances under $0.20/hr
truffle spot "c7i.*" --max-price 0.20 --sort-by-price

# Or Graviton alternatives
truffle spot "c8g.*" --max-price 0.20 --sort-by-price
```

### 5. Spot Interruption Planning

**Goal:** Identify backup AZs for Spot interruptions

```bash
# Get all AZs with the instance type
truffle az c7i.2xlarge --regions us-east-1

# Get current pricing in each AZ
truffle spot c7i.2xlarge --regions us-east-1

# Plan: Launch in cheapest AZ, failover to others
```

## 🔍 Understanding Spot Availability

### Current vs Spot Availability

**Important:** If an instance type is available for On-Demand, it's generally available for Spot too.

```bash
# Check On-Demand availability
truffle search m7i.large

# Check Spot pricing (if price exists, Spot is available)
truffle spot m7i.large
```

**No Spot pricing?** Could mean:
- ❌ No current Spot capacity in that AZ
- ❌ AWS temporarily disabled Spot for that type
- ✅ Try another AZ in the same region

### Spot Capacity Pools

Each AZ has **multiple Spot capacity pools** based on:
- Instance type
- Operating system (Linux/Windows)
- Network type

**Truffle shows Linux/UNIX pools** by default (most common).

## 💡 Spot Best Practices with Truffle

### 1. Maximize AZ Diversity

```bash
# ✅ Good: Find instances in 4+ AZs
truffle az m7i.large --min-az-count 4

# ❌ Bad: Only using 1 AZ
# Higher interruption risk!
```

### 2. Use Multiple Instance Types

```bash
# ✅ Good: Flexible with instance types
truffle spot "m7i.large" --regions us-east-1
truffle spot "m7i.xlarge" --regions us-east-1
truffle spot "m8g.large" --regions us-east-1  # Graviton alternative

# Configure Spot fleet with all three types
```

### 3. Monitor Price Changes

```bash
# Check prices daily
truffle spot m7i.large --regions us-east-1 --sort-by-price

# Compare to yesterday's prices
# Spot prices change based on supply/demand
```

### 4. Consider Graviton for Better Spot Availability

```bash
# Graviton instances often have:
# - Better Spot availability
# - Lower Spot prices
# - Fewer interruptions

truffle spot "m8g.xlarge" --show-savings
truffle spot "c8g.2xlarge" --show-savings
```

## 📋 Spot + AZ Strategy

### Complete Workflow

**Step 1: Find Available AZs**
```bash
truffle az m7i.large --regions us-east-1
```

**Step 2: Get Spot Pricing**
```bash
truffle spot m7i.large --regions us-east-1 --sort-by-price
```

**Step 3: Choose Top 3 Cheapest AZs**
```bash
# From output, select:
# - us-east-1c ($0.0298/hr)
# - us-east-1a ($0.0312/hr)
# - us-east-1d ($0.0354/hr)
```

**Step 4: Configure Spot Request**
```json
{
  "instance_type": "m7i.large",
  "availability_zones": [
    "us-east-1c",
    "us-east-1a", 
    "us-east-1d"
  ],
  "max_price": "0.04"
}
```

**Step 5: Launch with Diversity**
- Primary: us-east-1c (cheapest)
- Backup: us-east-1a, us-east-1d
- Auto-failover if interrupted

## 🎯 Advanced Spot Techniques

### Price History Analysis

```bash
# Check last 24 hours of pricing
truffle spot m7i.large --lookback-hours 24

# Compare to last week (168 hours)
truffle spot m7i.large --lookback-hours 168
```

### Multi-Region Spot Strategy

```bash
# Compare Spot prices across regions
truffle spot m7i.large --regions us-east-1 --output json > us-east-1.json
truffle spot m7i.large --regions us-west-2 --output json > us-west-2.json
truffle spot m7i.large --regions eu-west-1 --output json > eu-west-1.json

# Find globally cheapest option
cat *.json | jq -s 'add | sort_by(.spot_price) | .[0]'
```

### Spot Fleet Configuration

```bash
# Find multiple instance types for fleet
truffle spot "m7i.large" --regions us-east-1 --output json
truffle spot "m7i.xlarge" --regions us-east-1 --output json  
truffle spot "m8g.large" --regions us-east-1 --output json

# Use all in Spot Fleet for maximum flexibility
```

## 🔄 Handling Spot Interruptions

### Before Launch: Plan Failovers

```bash
# Get ALL AZs with instance type
truffle az m7i.large --regions us-east-1 --output json

# Result: 6 AZs available
# Configure auto-failover to all 6
```

### After Interruption: Find New AZ

```bash
# Quickly find alternative AZs
truffle spot m7i.large --regions us-east-1 --sort-by-price

# Launch in next-cheapest AZ
```

## 📊 Output Formats for Automation

### JSON for Scripts

```bash
truffle spot m7i.large --output json > spot-prices.json

# Parse with jq
cat spot-prices.json | jq '.[] | select(.spot_price < 0.05)'
```

### CSV for Spreadsheets

```bash
truffle spot "m7i.*" --output csv > spot-analysis.csv

# Open in Excel/Google Sheets for analysis
```

### Integration with Terraform

```bash
# Get cheapest AZ
CHEAPEST_AZ=$(truffle spot m7i.large --regions us-east-1 --output json | \
  jq -r 'sort_by(.spot_price) | .[0].availability_zone')

# Use in Terraform
terraform apply -var="spot_az=$CHEAPEST_AZ"
```

## ⚠️ Important Spot Considerations

### Spot ≠ Guaranteed Availability

- Spot instances can be interrupted with 2-minute warning
- No guarantees of capacity
- **Use Truffle to maximize availability** via AZ diversity

### Pricing Changes Frequently

- Spot prices fluctuate based on supply/demand
- Check prices regularly with `truffle spot`
- Set max price to avoid surprises

### Not All Instance Types Equal for Spot

**Better Spot availability:**
- ✅ Previous generation instances (m6i, c6i)
- ✅ Graviton instances (m8g, c8g, r8g)
- ✅ Common sizes (large, xlarge)

**Limited Spot availability:**
- ⚠️ Latest generation (first 3-6 months)
- ⚠️ Metal instances
- ⚠️ Specialized instances (GPU, accelerated)

Use Truffle to verify!

## 🎓 Real-World Example

### Scenario: Deploy 100-node Kubernetes cluster on Spot

**Requirements:**
- Instance: m8g.xlarge (Graviton)
- Region: us-east-1
- HA across AZs
- Target cost: <$0.10/hr per node

**Solution:**

```bash
# Step 1: Find AZ availability
truffle az m8g.xlarge --regions us-east-1 --min-az-count 3
# ✅ Available in 6 AZs

# Step 2: Get Spot pricing
truffle spot m8g.xlarge --regions us-east-1 --max-price 0.10 --sort-by-price
# ✅ Prices range $0.0567 - $0.0712

# Step 3: Select top 3 cheapest AZs
# - us-east-1c: $0.0567/hr
# - us-east-1a: $0.0594/hr
# - us-east-1f: $0.0612/hr

# Step 4: Configure EKS node groups
# - 34 nodes in us-east-1c
# - 33 nodes in us-east-1a
# - 33 nodes in us-east-1f

# Step 5: Monitor pricing
truffle spot m8g.xlarge --regions us-east-1 --sort-by-price
# Run daily to optimize
```

**Result:**
- ✅ 100 nodes across 3 AZs
- ✅ Average cost: $0.0591/hr per node
- ✅ Total: $5.91/hr for 100 nodes
- ✅ Savings: ~75% vs On-Demand!

## 📚 Summary

**Truffle for Spot Instances:**

✅ **Current Features (Already Useful!):**
- AZ availability checking
- Multi-AZ instance discovery
- AZ count filtering for diversity
- Region-wide searches

✅ **New Spot Features:**
- Real-time Spot pricing
- Price comparison across AZs
- Savings vs On-Demand
- Max price filtering
- Price-based sorting

✅ **Best Practices:**
- Use 3+ AZs for redundancy
- Check Spot prices regularly
- Consider Graviton for better availability
- Configure Spot fleets with multiple instance types
- Monitor and adapt to price changes

**Bottom Line:** Truffle helps you find the **cheapest, most available Spot instances** across all AWS regions and AZs! 🎯

---

**Quick Command Reference:**

```bash
# AZ diversity (current feature)
truffle az m7i.large --min-az-count 3

# Spot pricing (new feature!)
truffle spot m7i.large --sort-by-price

# Cheap Spot instances
truffle spot "m8g.*" --max-price 0.10

# Savings calculation
truffle spot m7i.xlarge --show-savings
```
