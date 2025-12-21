# GPU & ML Capacity Reservations Guide

Getting GPU instances (especially p5, g6, inf2, trn1) is **extremely difficult** due to high demand. On-Demand Capacity Reservations (ODCRs) are often **the only way** to guarantee access!

## 🎯 The GPU Shortage Problem

### Why GPU Instances Are Hard to Get

**Without Capacity Reservations:**
- ❌ p5.48xlarge: "InsufficientInstanceCapacity" error
- ❌ g6.xlarge: Hours of waiting, might never launch
- ❌ trn1.32xlarge: Not available in most AZs
- ❌ inf2.xlarge: Spotty availability

**With Capacity Reservations:**
- ✅ **Guaranteed** capacity when you need it
- ✅ Launch within seconds
- ✅ No "InsufficientInstanceCapacity" errors
- ✅ Plan ML training runs with confidence

## 🚀 Quick Start

### Check GPU Capacity Reservations

```bash
# Check all GPU reservations
truffle capacity --gpu-only

# Check specific GPU instances
truffle capacity --instance-types p5.48xlarge,g6.xlarge

# Only show reservations with available capacity
truffle capacity --available-only --min-capacity 1

# Find reservations in specific regions
truffle capacity --gpu-only --regions us-east-1,us-west-2
```

### Example Output

```
📊 Capacity Reservation Summary:
   Total Reservations: 12
   Active Reservations: 12
   Instance Types: 4 (p5.48xlarge, g6.xlarge, inf2.xlarge, trn1.2xlarge)
   Regions: 3
   Availability Zones: 8
   Total Capacity: 156 instances
   Available Capacity: 48 instances (30.8% free)
   Used Capacity: 108 instances (69.2% utilized)

+----------------+-----------+-------------+-------+-----------+------+--------+----------------+
| Instance Type  | Region    | AZ          | Total | Available | Used | State  | Reservation ID |
+----------------+-----------+-------------+-------+-----------+------+--------+----------------+
| p5.48xlarge    | us-east-1 | us-east-1a | 32    | 12        | 20   | active | cr-0ab123...   |
|                |           | us-east-1c | 16    | 8         | 8    | active | cr-0cd456...   |
| g6.xlarge      | us-west-2 | us-west-2a | 64    | 24        | 40   | active | cr-0ef789...   |
| inf2.xlarge    | eu-west-1 | eu-west-1a | 32    | 4         | 28   | active | cr-0gh012...   |
| trn1.2xlarge   | us-east-1 | us-east-1b | 12    | 0         | 12   | active | cr-0ij345...   |
+----------------+-----------+-------------+-------+-----------+------+--------+----------------+
```

## 💡 Understanding ODCRs (On-Demand Capacity Reservations)

### What are ODCRs?

**ODCRs = Reserved capacity slots** that guarantee you can launch instances:
- Reserve specific instance types in specific AZs
- Pay On-Demand prices (no upfront cost)
- Can be used immediately or left for future use
- Billed whether used or not (once created)

### How They Work

1. **Create ODCR**: Reserve N instances in an AZ
   - Example: 16x p5.48xlarge in us-east-1a
   
2. **Launch Instances**: Use reserved capacity
   - Launches immediately (no capacity errors!)
   - Uses ODCR allocation automatically
   
3. **Pay for Usage**: 
   - ODCR itself: No charge (just a reservation)
   - Instances launched: Normal On-Demand price
   - Unused capacity: You're NOT charged if unused

### ODCR vs Savings Plans vs Reserved Instances

| Feature | ODCR | Savings Plans | Reserved Instances |
|---------|------|---------------|-------------------|
| Guarantees capacity | ✅ YES | ❌ No | ✅ Yes |
| Upfront payment | ❌ No | ✅ Yes | ✅ Yes |
| Flexible | ✅ Yes | ⚠️ Partial | ❌ No |
| Best for GPUs | ✅ YES | ❌ No | ⚠️ Limited |
| Can release | ✅ Yes | ❌ No | ⚠️ Can sell |

**For ML/GPU workloads:** ODCRs are the best option!

## 🎮 GPU Instance Types (December 2025)

### Latest GPU Instances

**NVIDIA GPUs:**
- **p5.48xlarge** - 8x H100 (80GB each) - **Most in-demand!**
- **p4d.24xlarge** - 8x A100 (40GB each)
- **g6.xlarge** to **g6.48xlarge** - L4/L40S GPUs
- **g5.xlarge** to **g5.48xlarge** - A10G GPUs

**AWS ML Accelerators:**
- **inf2.xlarge** to **inf2.48xlarge** - Inferentia2 (inference)
- **trn1.2xlarge** to **trn1.32xlarge** - Trainium (training)
- **trn1n.32xlarge** - Trainium with enhanced networking

### Availability by Instance Type

| Instance | Availability | ODCR Needed? | Typical Use |
|----------|--------------|--------------|-------------|
| **p5.48xlarge** | ⚠️ Very Low | ✅ CRITICAL | Large LLM training |
| **p4d.24xlarge** | ⚠️ Low | ✅ Recommended | LLM training/fine-tuning |
| **g6.xlarge** | ⚠️ Medium | ✅ Recommended | Video, graphics, small ML |
| **inf2.xlarge** | ✅ Good | ⚠️ Optional | LLM inference |
| **trn1.32xlarge** | ⚠️ Low | ✅ Recommended | Large model training |

## 📊 Finding Available Capacity

### 1. Check Existing ODCRs (Shared/Public)

```bash
# Find all GPU ODCRs with available capacity
truffle capacity --gpu-only --available-only

# Specific instances
truffle capacity --instance-types p5.48xlarge --available-only

# In your region
truffle capacity --gpu-only --regions us-east-1 --available-only
```

**What this shows:**
- ODCRs created by your organization
- Shared ODCRs (if any)
- Current utilization

### 2. Check Which AZs Support GPU Instances

```bash
# Find which AZs have p5.48xlarge
truffle search p5.48xlarge

# Find AZs with at least 3 AZs (for redundancy)
truffle az p5.48xlarge --min-az-count 3

# Check multiple GPU types
truffle search "p5.*" --regions us-east-1
truffle search "g6.*" --regions us-west-2
```

### 3. Check Instance Type Availability

```bash
# See if GPU instances exist in a region at all
truffle search p5.48xlarge --regions us-east-1,us-west-2,eu-west-1
```

**If NOT available in any AZ:**
- Instance type not supported in that region
- Try different regions

**If available in some AZs:**
- You can request ODCRs in those AZs
- Create ODCR via AWS Console/CLI

## 🔍 Real-World Scenarios

### Scenario 1: Training Large Language Model

**Requirements:**
- Instance: p5.48xlarge (8x H100)
- Duration: 2 weeks
- Must complete by deadline

**Solution:**

```bash
# Step 1: Check which AZs have p5.48xlarge
truffle search p5.48xlarge --regions us-east-1

# Output shows: Available in us-east-1a, us-east-1c

# Step 2: Check existing ODCRs
truffle capacity --instance-types p5.48xlarge --regions us-east-1

# Output shows: 12 available in us-east-1a

# Step 3: Launch training job
# Use the AZ with available ODCR capacity
# → Launch in us-east-1a (guaranteed capacity!)
```

**Result:** ✅ Training starts immediately, no capacity errors!

### Scenario 2: ML Inference Service

**Requirements:**
- Instance: inf2.xlarge (AWS Inferentia2)
- Auto-scaling: 10-100 instances
- 24/7 availability

**Solution:**

```bash
# Step 1: Find AZs with inf2.xlarge
truffle az inf2.xlarge --min-az-count 3 --regions us-east-1

# Output: Available in us-east-1a, 1b, 1c, 1d, 1e, 1f

# Step 2: Check existing capacity
truffle capacity --instance-types inf2.xlarge --available-only

# Step 3: Create ODCRs if needed
# Reserve 100 instances across 3 AZs
# - 34 in us-east-1a
# - 33 in us-east-1b  
# - 33 in us-east-1c
```

**Result:** ✅ Can scale to 100 instances instantly!

### Scenario 3: Multi-Region GPU Deployment

**Requirements:**
- Instance: g6.xlarge
- Regions: us-east-1, us-west-2, eu-west-1
- Need capacity guarantee in all regions

**Solution:**

```bash
# Step 1: Check all regions
truffle search g6.xlarge --regions us-east-1,us-west-2,eu-west-1

# Step 2: Check ODCRs in each region
truffle capacity --instance-types g6.xlarge --regions us-east-1
truffle capacity --instance-types g6.xlarge --regions us-west-2
truffle capacity --instance-types g6.xlarge --regions eu-west-1

# Step 3: Find gaps
# If us-west-2 has no ODCRs, create them there

# Step 4: Get AZ details for ODCR creation
truffle az g6.xlarge --regions us-west-2
```

### Scenario 4: Monitoring GPU Capacity

**Goal:** Alert when GPU capacity becomes available

```bash
# Monitor script (run every 5 minutes)
#!/bin/bash

PREV_COUNT=$(cat /tmp/gpu-capacity.txt 2>/dev/null || echo 0)
CURRENT_COUNT=$(truffle capacity --gpu-only --available-only --output json | jq 'length')

if [ "$CURRENT_COUNT" -gt "$PREV_COUNT" ]; then
    echo "🎉 New GPU capacity available!"
    truffle capacity --gpu-only --available-only
fi

echo "$CURRENT_COUNT" > /tmp/gpu-capacity.txt
```

## 🛠️ Using as a Go Module

### Monitor GPU Capacity in Your Application

```go
package main

import (
    "context"
    "fmt"
    "time"
    "github.com/yourusername/truffle/pkg/aws"
)

func monitorGPUCapacity() {
    ctx := context.Background()
    client, _ := aws.NewClient(ctx)
    
    gpuTypes := []string{"p5.48xlarge", "g6.xlarge"}
    regions := []string{"us-east-1", "us-west-2"}
    
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        reservations, err := client.GetCapacityReservations(ctx, regions, 
            aws.CapacityReservationOptions{
                InstanceTypes: gpuTypes,
                OnlyAvailable: true,
                MinCapacity:   1,
            })
        
        if err != nil {
            continue
        }
        
        for _, r := range reservations {
            fmt.Printf("🎮 Available: %d x %s in %s\n",
                r.AvailableCapacity, r.InstanceType, r.AvailabilityZone)
            
            // Send alert, update database, etc.
            sendAlert(r)
        }
    }
}
```

### Pre-Launch Validation

```go
func validateGPUCapacity(instanceType, region string, count int32) error {
    client, _ := aws.NewClient(context.Background())
    
    reservations, err := client.GetCapacityReservations(ctx, []string{region},
        aws.CapacityReservationOptions{
            InstanceTypes: []string{instanceType},
            OnlyAvailable: true,
            MinCapacity:   count,
        })
    
    if err != nil {
        return err
    }
    
    if len(reservations) == 0 {
        return fmt.Errorf("no capacity available for %s in %s", 
            instanceType, region)
    }
    
    return nil
}
```

## 📋 Best Practices

### 1. Always Check Before Large ML Runs

```bash
# Before starting expensive training
truffle capacity --instance-types p5.48xlarge --available-only

# If no capacity → Request ODCR first
# Don't risk "InsufficientInstanceCapacity" mid-training!
```

### 2. Use Multiple AZs for Redundancy

```bash
# Create ODCRs in 2-3 AZs
truffle az p5.48xlarge --min-az-count 3

# If one AZ has issues, use another
```

### 3. Monitor Utilization

```bash
# Check ODCR utilization weekly
truffle capacity --gpu-only --output json > capacity-report.json

# Identify underutilized reservations
# Consider releasing if not needed
```

### 4. Plan Ahead for New Instances

```bash
# New instance type announced?
truffle search p6.* --regions us-east-1

# Monitor when it becomes available
# Request ODCR as soon as possible
```

## ⚠️ Common Issues

### "InsufficientInstanceCapacity"

**Problem:** Can't launch GPU instance

**Solution:**
```bash
# Check if ODCRs exist
truffle capacity --instance-types p5.48xlarge --available-only

# If none → Create ODCR
# If exists → Use that specific AZ when launching
```

### "No Capacity Reservations Found"

**Problem:** No ODCRs in your account

**Solution:**
```bash
# Check if instance type is available at all
truffle search p5.48xlarge --regions us-east-1

# If available → Create ODCR via AWS Console/CLI
# If not available → Try different region
```

### "Used Capacity = Total Capacity"

**Problem:** ODCR is fully utilized

**Options:**
1. Wait for instances to terminate
2. Create additional ODCR
3. Use different AZ
4. Use different instance type

```bash
# Find alternative AZs
truffle az p5.48xlarge --regions us-east-1

# Check other AZs for capacity
truffle capacity --instance-types p5.48xlarge --available-only
```

## 📊 Integration with ML Workflows

### Terraform

```hcl
# Use Truffle to find AZ with capacity
data "external" "gpu_capacity" {
  program = ["bash", "-c", <<-EOF
    truffle capacity \
      --instance-types p5.48xlarge \
      --available-only \
      --output json | \
      jq -r '.[0] | {az: .availability_zone, region: .region}'
  EOF
  ]
}

resource "aws_instance" "gpu_training" {
  instance_type     = "p5.48xlarge"
  availability_zone = data.external.gpu_capacity.result.az
  # ... rest of config
}
```

### Kubernetes

```yaml
# Node affinity based on ODCR availability
apiVersion: v1
kind: Pod
metadata:
  name: ml-training
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: topology.kubernetes.io/zone
            operator: In
            values:
            # Get from: truffle capacity --gpu-only --available-only
            - us-east-1a
            - us-east-1c
  # ... rest of pod spec
```

### Ray/Distributed Training

```python
# Check capacity before Ray cluster launch
import subprocess
import json

def get_available_gpu_azs(instance_type, region):
    result = subprocess.run([
        'truffle', 'capacity',
        '--instance-types', instance_type,
        '--regions', region,
        '--available-only',
        '--output', 'json'
    ], capture_output=True, text=True)
    
    reservations = json.loads(result.stdout)
    return [r['availability_zone'] for r in reservations]

# Use for Ray head node placement
azs = get_available_gpu_azs('p5.48xlarge', 'us-east-1')
print(f"Launch Ray head in: {azs[0]}")
```

## 🎯 Quick Command Reference

```bash
# Check all GPU ODCRs
truffle capacity --gpu-only

# Find available GPU capacity
truffle capacity --gpu-only --available-only

# Specific GPU instance
truffle capacity --instance-types p5.48xlarge

# Minimum capacity needed
truffle capacity --gpu-only --min-capacity 10

# Export for analysis
truffle capacity --gpu-only --output json > gpu-capacity.json

# Monitor (cron job)
*/5 * * * * truffle capacity --gpu-only --available-only | mail -s "GPU Capacity" you@email.com
```

## 📚 Summary

**ODCRs are ESSENTIAL for GPU/ML workloads because:**
- ✅ Guarantee capacity (no "InsufficientInstanceCapacity")
- ✅ Launch instantly (critical for ML deadlines)
- ✅ Plan with confidence (know capacity is reserved)
- ✅ No upfront cost (pay only when you use)

**Use Truffle to:**
- 🔍 Find existing ODCRs
- 📊 Monitor capacity utilization
- 🎯 Identify which AZs to create new ODCRs
- ⚡ Automate capacity checks in your ML pipelines

**For in-demand GPU instances (p5, g6, trn1), ODCRs are often THE ONLY WAY to get capacity!**

---

**Start checking capacity:**
```bash
truffle capacity --gpu-only --available-only
```

See **MODULE_USAGE.md** for Go library integration examples!
