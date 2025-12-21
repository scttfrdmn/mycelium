# ✅ Spot Instance Support - What's New

Great question! Yes, Truffle is **super useful for Spot instances** - and now even better with dedicated Spot pricing features!

## 🎯 The Answer: Two-Part Value

### Part 1: **Already Useful** (Existing Features)

Truffle's AZ-focused approach is **perfect** for Spot instances because:

✅ **AZ Diversity = Better Spot Availability**
- Spot works best when you can launch across multiple AZs
- Truffle shows you **which AZs have your instance type**
- More AZs = better chance of getting Spot capacity

✅ **Spot Capacity Varies by AZ**
- Same instance type may be available in some AZs but not others
- Truffle reveals this **AZ-level availability** by default

**Example:**
```bash
# Find instances available in 3+ AZs (critical for Spot!)
truffle az m7i.large --min-az-count 3

# Output shows ALL AZs with this instance
# → Use these for Spot fleet diversification
```

### Part 2: **NEW! Spot Pricing** 🆕

I've added a dedicated `truffle spot` command that shows:

✅ **Real-time Spot prices** across all AZs  
✅ **Price comparisons** to find cheapest AZs  
✅ **Savings calculations** vs On-Demand  
✅ **Price filtering** to find budget-friendly options  

**Example:**
```bash
# Get current Spot prices
truffle spot m7i.large

# Find Spot instances under $0.10/hour
truffle spot "m8g.*" --max-price 0.10 --sort-by-price

# Show savings vs On-Demand
truffle spot m7i.xlarge --show-savings
```

## 📊 Extra Steps for Spot Instances?

### Using Existing Commands (No extra steps!)

```bash
# Step 1: Find AZ availability (for diversity)
truffle search m7i.large
# Shows: Available in us-east-1a, 1b, 1c, 1d, 1e, 1f

# Step 2: That's it! Use those AZs for Spot
```

**No extra AWS API calls needed** - if an instance is available On-Demand in an AZ, it's generally available for Spot too!

### Using NEW Spot Command (Extra info!)

```bash
# Step 1: Check AZ availability
truffle az m7i.large --min-az-count 3

# Step 2: Get Spot pricing
truffle spot m7i.large --sort-by-price

# Step 3: Choose cheapest AZs for Spot fleet
```

**Extra API call** (`DescribeSpotPriceHistory`) but gives you:
- Current Spot prices per AZ
- Which AZs are cheapest
- Potential savings

## 🆕 New Files Added

### 1. **cmd/spot.go** - Spot Command
New `truffle spot` command with:
- Real-time Spot pricing
- Price filtering (`--max-price`)
- Savings calculations (`--show-savings`)
- Price-based sorting (`--sort-by-price`)

### 2. **SPOT_GUIDE.md** - Complete Guide
Comprehensive documentation covering:
- Why AZ diversity matters for Spot
- How to use Spot pricing data
- Multi-AZ Spot fleet strategies
- Kubernetes Spot node groups
- Cost optimization techniques
- Handling Spot interruptions
- Real-world examples

### 3. **pkg/aws/client.go** - Updated
Added:
- `SpotPriceResult` struct
- `SpotOptions` struct  
- `GetSpotPricing()` method
- Spot price history queries

### 4. **pkg/output/printer.go** - Updated
Added Spot output methods:
- `PrintSpotTable()` - Formatted table
- `PrintSpotJSON()` - JSON output
- `PrintSpotYAML()` - YAML output
- `PrintSpotCSV()` - CSV output

### 5. **README.md** - Updated
Added Spot command documentation and examples

## 💡 Why This Matters for Spot

### Problem: Spot Interruptions
**Solution:** Use Truffle to find 4-6 AZs with your instance type

```bash
truffle az m7i.large --min-az-count 4
# Configure Spot fleet with all AZs
# → Better availability, less interruptions
```

### Problem: Spot Price Volatility
**Solution:** Use Truffle to find cheapest AZs

```bash
truffle spot m7i.large --sort-by-price
# Launch in top 3 cheapest AZs
# → Lower costs, better value
```

### Problem: Capacity Planning
**Solution:** Use Truffle to verify AZ+region availability

```bash
# Check multiple regions
truffle spot m7i.large --regions us-east-1,us-west-2,eu-west-1

# Find globally cheapest option
truffle spot m7i.large --output json | jq 'sort_by(.spot_price) | .[0]'
```

## 🎯 Common Spot Workflows

### Workflow 1: Find Best Spot Instances

```bash
# Find all Graviton instances under $0.15/hr
truffle spot "m8g.*" --max-price 0.15 --sort-by-price

# Compare savings
truffle spot m8g.xlarge --show-savings
# Output: "87% savings vs On-Demand!"
```

### Workflow 2: Multi-AZ Kubernetes

```bash
# Find instance in 3+ AZs
truffle az m8g.xlarge --min-az-count 3 --regions us-west-2

# Get Spot prices for each AZ
truffle spot m8g.xlarge --regions us-west-2 --sort-by-price

# Configure EKS node groups with top 3 AZs
```

### Workflow 3: Batch Processing

```bash
# Find compute instances under $0.20/hr
truffle spot "c7i.*" --max-price 0.20 --sort-by-price

# Use cheapest for batch jobs
# Auto-failover to next cheapest on interruption
```

## 📊 Output Examples

### Spot Pricing Table

```bash
$ truffle spot m7i.large --regions us-east-1

💰 Spot Instance Summary:
   Instance Types: 1
   Regions: 1
   Availability Zones: 6
   Price Range: $0.0298 - $0.0512 per hour
   Average Price: $0.0387 per hour

+---------------+-----------+------------------+----------------+
| Instance Type | Region    | Availability Zone| Spot Price/hr  |
+---------------+-----------+------------------+----------------+
| m7i.large     | us-east-1 | us-east-1c      | $0.0298        |← Cheapest!
|               |           | us-east-1a      | $0.0312        |
|               |           | us-east-1d      | $0.0354        |
|               |           | us-east-1f      | $0.0389        |
|               |           | us-east-1b      | $0.0405        |
|               |           | us-east-1e      | $0.0512        |
+---------------+-----------+------------------+----------------+
```

### With Savings

```bash
$ truffle spot m7i.large --show-savings --regions us-east-1

+---------------+-----------+---------+--------------+---------------+----------+
| Instance Type | Region    | AZ      | Spot/hr      | On-Demand/hr  | Savings  |
+---------------+-----------+---------+--------------+---------------+----------+
| m7i.large     | us-east-1 | 1c      | $0.0298      | $0.1248       | 76.1%    |
|               |           | 1a      | $0.0312      | $0.1248       | 75.0%    |
+---------------+-----------+---------+--------------+---------------+----------+
```

### JSON for Automation

```bash
$ truffle spot m7i.large --output json

[
  {
    "instance_type": "m7i.large",
    "region": "us-east-1",
    "availability_zone": "us-east-1c",
    "spot_price": 0.0298,
    "on_demand_price": 0.1248,
    "savings_percent": 76.1,
    "timestamp": "2025-12-17T12:00:00Z"
  }
]
```

## 🔑 Key Takeaways

### Question: "Is Truffle useful for Spot?"
**Answer: YES! Very useful!** ✅

**Why:**
1. Shows AZ availability (critical for Spot diversity)
2. Reveals which specific AZs have capacity
3. NEW: Shows Spot pricing across AZs
4. Helps find cheapest Spot instances
5. Enables multi-AZ Spot fleet planning

### Question: "Is there an extra step for Spot?"
**Answer: No extra step required, but optional Spot pricing available!**

**Basic (existing features):**
```bash
truffle search m7i.large
# Shows AZ availability - use for Spot!
```

**Advanced (NEW Spot features):**
```bash
truffle spot m7i.large --sort-by-price
# Shows Spot prices - find cheapest AZs!
```

## 🚀 Summary

**What you get:**

✅ **AZ-level availability** - perfect for Spot diversity (existing)  
✅ **Spot pricing data** - find cheapest AZs (NEW!)  
✅ **Savings calculations** - see cost benefits (NEW!)  
✅ **Multi-region comparison** - global Spot optimization (NEW!)  
✅ **Automation-friendly** - JSON/CSV output for scripts  

**Perfect for:**
- Kubernetes Spot node groups
- Batch processing workloads  
- CI/CD on Spot instances
- Development environments
- Cost optimization projects

**Bottom line:** Truffle is now a **complete Spot instance tool** - from finding available AZs to getting real-time pricing! 🎯

---

**Quick Reference:**

```bash
# AZ availability (for Spot diversity)
truffle az m7i.large --min-az-count 3

# Spot pricing (for cost optimization)
truffle spot m7i.large --sort-by-price

# Combined workflow
truffle az m7i.large --min-az-count 3  # Find AZs
truffle spot m7i.large --sort-by-price # Get prices
# → Configure Spot fleet with top 3 cheapest AZs!
```
