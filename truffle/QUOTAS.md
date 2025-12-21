# AWS Service Quotas Support in truffle

## 🎯 The Problem

```bash
# User finds perfect instance
truffle search m7i.xlarge

# Pipes to spawn
truffle search m7i.xlarge | spawn

# Launch fails! 😱
❌ Error: VcpuLimitExceeded
   You have requested more vCPU capacity than your current limit
```

**User is confused and frustrated!**

## ✅ The Solution

```bash
# Check quotas first
truffle search m7i.xlarge --check-quotas

✅ QUOTAS OK
   Available: 28 vCPUs (Standard family)
   m7i.xlarge needs: 4 vCPUs

# Or get detailed view
truffle quotas

# Now launch with confidence!
truffle search m7i.xlarge --check-quotas | spawn
```

---

## 🔑 AWS Credentials Required

### Why Credentials Are Needed

Quota checking requires AWS credentials because it:
- Queries **Service Quotas API** (reads your account limits)
- Queries **EC2 API** (counts running instances)
- Calculates available capacity

### Graceful Degradation

```bash
# Without credentials (works!)
truffle search m7i.large
# Shows instance types from public data
# No quota checking

# With credentials (enhanced!)
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
truffle search m7i.large --check-quotas
# Shows instances + quota availability
```

**truffle remains usable without credentials!**

---

## 📊 Understanding AWS Quotas

### What Are Service Quotas?

AWS limits how many resources you can use to:
- Prevent accidental large bills
- Manage capacity
- Prevent abuse

### Key EC2 Quotas

#### 1. **On-Demand vCPU Limits** (Most Important)

AWS groups instances by family:

| Family | Instance Types | Default Limit | Notes |
|--------|---------------|---------------|-------|
| **Standard** | A, C, D, H, I, M, R, T, Z | 5-32 vCPUs | Most common instances |
| **G** | g4dn, g5, g6 | 0-128 vCPUs | Graphics/GPU |
| **P** | p3, p4, p5 | 0 vCPUs | GPU training (requires request) |
| **Inf** | inf1, inf2 | 0 vCPUs | Inferentia (requires request) |
| **Trn** | trn1 | 0 vCPUs | Trainium (requires request) |
| **F** | f1 | 0-8 vCPUs | FPGA |
| **X** | x1, x2 | 0-128 vCPUs | Memory optimized |

**Critical:** GPU quotas (P, G, Inf, Trn) are often **0 by default!**

#### 2. **Spot vCPU Limits**

Separate quotas for Spot instances (same families).

#### 3. **Other Limits**

- Running instances: 20 (default)
- VPCs per region: 5
- Elastic IPs: 5
- Security groups: 500

---

## 🚀 Usage Examples

### Example 1: Basic Quota Check

```bash
$ truffle search m7i.large --check-quotas

╔════════════════════════════════════════════════════════╗
║  📊 AWS Service Quotas Summary                        ║
╚════════════════════════════════════════════════════════╝

Region: us-east-1

  ✅ Standard: 4/32 vCPUs (available: 28)
  ✅ G: 0/128 vCPUs (available: 128)
  🔴 P: 0/0 vCPUs (available: 0)

✅ Instances You Can Launch:

┌──────────────────────────────────────────────────┐
│ Instance Type: m7i.large                         │
│ vCPUs: 2, Memory: 8 GiB                         │
│ ✅ Can launch (2/28 vCPUs available)            │
└──────────────────────────────────────────────────┘
```

### Example 2: Quota Exceeded

```bash
$ truffle search m7i.xlarge --check-quotas

╔════════════════════════════════════════════════════════╗
║  📊 AWS Service Quotas Summary                        ║
╚════════════════════════════════════════════════════════╝

Region: us-east-1

  🟡 Standard: 30/32 vCPUs (available: 2)

⚠️  1 instance(s) filtered due to quota limits
   Use --show-all to see blocked instances

# No results shown (filtered out)
```

### Example 3: Show All (Including Blocked)

```bash
$ truffle search m7i.xlarge --check-quotas --show-all

⚠️  Instances Blocked by Quota:

┌──────────────────────────────────────────────────┐
│ m7i.xlarge                                       │
│ vCPUs: 4, Memory: 16 GiB                        │
│                                                  │
│ ❌ QUOTA EXCEEDED                                │
│ Need 4 vCPUs, only 2 available                  │
│ (quota: 32, usage: 30)                           │
│                                                  │
│ 💡 To launch this instance:                      │
│    Request quota increase for Standard family   │
└──────────────────────────────────────────────────┘

# Request Standard quota increase to 64 vCPUs
aws service-quotas request-service-quota-increase \
  --service-code ec2 \
  --quota-code L-1216C47A \
  --desired-value 64 \
  --region us-east-1
```

### Example 4: GPU Instance (Zero Quota)

```bash
$ truffle capacity --instance-types p5.48xlarge --check-quotas

╔════════════════════════════════════════════════════════╗
║  📊 AWS Service Quotas Summary                        ║
╚════════════════════════════════════════════════════════╝

Region: us-east-1

  🔴 P: 0/0 vCPUs (available: 0)

❌ No instances available within your quota limits

💡 Options:
   1. Request quota increase (P family requires 192 vCPUs for p5.48xlarge)
   2. Use smaller GPU instance (g5.xlarge has quota)
   3. Try different region

# Request P quota increase to 192 vCPUs
aws service-quotas request-service-quota-increase \
  --service-code ec2 \
  --quota-code L-417A185B \
  --desired-value 192 \
  --region us-east-1 \
  --description "ML training with H100 GPUs"
```

---

## 📋 Dedicated Quota Command

### View All Quotas

```bash
$ truffle quotas

╔════════════════════════════════════════════════════════╗
║  📊 AWS Service Quotas - us-east-1                    ║
╚════════════════════════════════════════════════════════╝

┌──────────┬────────────┬────────────┬───────┬────────────┬────────┐
│ Family   │ Type       │ Quota      │ Usage │ Available  │ Status │
├──────────┼────────────┼────────────┼───────┼────────────┼────────┤
│ Standard │ On-Demand  │ 32 vCPUs   │ 4     │ 28 vCPUs   │ ✅ OK  │
│ Standard │ Spot       │ 32 vCPUs   │ -     │ 32 vCPUs   │ ✅ OK  │
│ G        │ On-Demand  │ 128 vCPUs  │ 0     │ 128 vCPUs  │ ✅ OK  │
│ G        │ Spot       │ 128 vCPUs  │ -     │ 128 vCPUs  │ ✅ OK  │
│ P        │ On-Demand  │ 0 vCPUs    │ 0     │ 0 vCPUs    │ ❌ Zero│
│ Inf      │ On-Demand  │ 0 vCPUs    │ 0     │ 0 vCPUs    │ ❌ Zero│
└──────────┴────────────┴────────────┴───────┴────────────┴────────┘

🖥️  Running Instances: 2 / 20

📚 Instance Family Reference:
   Standard: A, C, D, H, I, M, R, T, Z (general purpose)
   G: Graphics/GPU instances (g4dn, g5, g6)
   P: GPU training instances (p3, p4, p5)
   Inf: Inferentia instances (inf1, inf2)
   Trn: Trainium instances (trn1)
```

### Filter by Family

```bash
$ truffle quotas --family P

# Shows only P family quotas
```

### Generate Increase Requests

```bash
$ truffle quotas --family P --request

╔════════════════════════════════════════════════════════╗
║  📝 Quota Increase Request Commands                   ║
╚════════════════════════════════════════════════════════╝

# us-east-1 - P Family
# Request P On-Demand quota increase to 192 vCPUs
aws service-quotas request-service-quota-increase \
  --service-code ec2 \
  --quota-code L-417A185B \
  --desired-value 192 \
  --region us-east-1

# Check status:
aws service-quotas list-requested-service-quota-change-history-by-quota \
  --service-code ec2 \
  --quota-code L-417A185B \
  --region us-east-1

💡 Notes:
   • Quota increases typically approved within 24-48 hours
   • GPU quotas (P, G, Inf, Trn) require business justification
   • Include your use case for faster approval
```

### Multi-Region Quotas

```bash
$ truffle quotas --regions us-east-1,us-west-2,eu-west-1

# Shows quotas for all three regions
# Useful for finding regions with available capacity
```

---

## 🔄 Integration with spawn

### Filtered Pipeline

```bash
# Only passes launchable instances to spawn
truffle search m7i.large --check-quotas | spawn

# If quota exceeded, spawn receives nothing
# No launch attempt, no error!
```

### Manual Override

```bash
# Show all, let user decide
truffle search m7i.large --check-quotas --show-all

# User sees warnings but can still try
# Maybe they'll free up quota first
```

---

## 💡 Best Practices

### For Beginners

1. **Always check quotas** when starting:
   ```bash
   truffle quotas
   ```

2. **Use --check-quotas** before launching:
   ```bash
   truffle search <type> --check-quotas | spawn
   ```

3. **Request GPU quotas early** (they're 0 by default):
   ```bash
   truffle quotas --family P --request
   ```

### For Power Users

1. **Enable by default** in config:
   ```toml
   # ~/.truffle/config.toml
   [search]
   check_quotas = true
   ```

2. **Multi-region quota comparison**:
   ```bash
   truffle quotas --regions us-east-1,us-west-2 > quotas.txt
   # Find region with best availability
   ```

3. **Automate quota monitoring**:
   ```bash
   #!/bin/bash
   # quota-alert.sh
   truffle quotas --regions us-east-1 | grep "🔴"
   if [ $? -eq 0 ]; then
       echo "ALERT: Quotas critically low!"
   fi
   ```

---

## 🎓 Common Scenarios

### Scenario 1: First GPU Instance

```bash
# User: "I want to try p5.48xlarge"
$ truffle capacity --instance-types p5.48xlarge --check-quotas

❌ P quota is 0 (request increase)

# User requests quota
$ truffle quotas --family P --request
# Copy/paste AWS command
# Wait 24-48 hours

# After approval
$ truffle capacity --instance-types p5.48xlarge --check-quotas
✅ Can launch (192/192 vCPUs available)
```

### Scenario 2: Running Out of Quota

```bash
# User has many instances running
$ truffle quotas

🟡 Standard: 28/32 vCPUs (available: 4)

# Can't launch large instance
$ truffle search m7i.2xlarge --check-quotas
⚠️  Need 8 vCPUs, only 4 available

# Options:
# 1. Stop some instances
# 2. Request quota increase
# 3. Use smaller instance
```

### Scenario 3: Multi-Region Deployment

```bash
# Find region with best quota
$ truffle quotas --regions us-east-1,us-west-2,eu-west-1

# us-east-1: 2/32 vCPUs available
# us-west-2: 28/32 vCPUs available ✅ (best!)
# eu-west-1: 15/32 vCPUs available

# Launch in us-west-2
$ truffle search m7i.xlarge --regions us-west-2 --check-quotas | spawn
```

---

## 🔧 Implementation Details

### Quota Families

```go
// Instance type → Quota family mapping
p5.48xlarge  → P (GPU Training)
g6.xlarge    → G (Graphics)
m7i.large    → Standard
inf2.xlarge  → Inf (Inferentia)
trn1.32xlarge → Trn (Trainium)
f1.2xlarge   → F (FPGA)
x2iedn.32xlarge → X (Memory)
```

### API Calls

1. **Service Quotas API**: Get limits
   - `GetServiceQuota` for each family
   - Cached for 5 minutes

2. **EC2 API**: Get usage
   - `DescribeInstances` (running + pending)
   - Count vCPUs by family

3. **Calculation**:
   ```
   Available = Quota - Usage
   Can Launch = (Instance vCPUs <= Available)
   ```

### Caching

- Quotas cached for 5 minutes
- Reduces API calls
- Faster repeated queries

---

## ⚠️ Important Notes

### Credentials Optional

- truffle works **without credentials**
- `--check-quotas` requires credentials
- Gracefully degrades if unavailable

### Quota Types

- **Applied**: Effective quota (what you have)
- **Requested**: Pending increases
- **Default**: Starting quota (often 0 for GPU)

### Regional Differences

- Quotas are **per-region**
- us-east-1 often has higher defaults
- New regions may have lower quotas

### Spot vs On-Demand

- **Separate quotas** for Spot
- Spot quotas usually higher
- Can't mix (using Spot doesn't affect On-Demand quota)

---

## 🎯 Impact

### For Beginners
- ✅ **No surprise failures** ("why can't I launch?")
- ✅ **Educational** (learn about quotas)
- ✅ **Actionable** (clear steps to fix)
- ✅ **Proactive** (check before launch)

### For Power Users
- ✅ **Time-saving** (no failed launches)
- ✅ **Automation-friendly** (programmatic checking)
- ✅ **Multi-region** (find best availability)
- ✅ **Planning** (know limits in advance)

### For ML Engineers
- ✅ **GPU-critical** (GPU quotas often 0)
- ✅ **Expensive failures avoided** (don't waste time)
- ✅ **Batch planning** ("can I launch 10x p5?")

---

## 📊 Feature Summary

| Feature | Command | Credentials | Notes |
|---------|---------|-------------|-------|
| Search without quotas | `truffle search` | ❌ No | Works for everyone |
| Search with quotas | `truffle search --check-quotas` | ✅ Yes | Filters results |
| View all quotas | `truffle quotas` | ✅ Yes | Detailed view |
| Request increases | `truffle quotas --request` | ✅ Yes | Generates commands |
| Multi-region | `truffle quotas --regions ...` | ✅ Yes | Compare regions |

---

## 🚀 Next Steps

1. **Configure AWS credentials**:
   ```bash
   aws configure
   ```

2. **Check your quotas**:
   ```bash
   truffle quotas
   ```

3. **Request GPU quota** (if needed):
   ```bash
   truffle quotas --family P --request
   ```

4. **Use quota checking**:
   ```bash
   truffle search <type> --check-quotas | spawn
   ```

**Now you'll never hit quota limits unexpectedly!** ✨
