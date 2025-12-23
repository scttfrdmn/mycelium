# spawn - Ephemeral AWS EC2 Instance Launcher

**spawn** makes launching AWS EC2 instances effortless and foolproof. Perfect companion to [truffle](../truffle).

## 🎯 Philosophy

spawn is designed for **non-experts** who just need compute:
- ✅ Launch instances in seconds, not hours
- ✅ Auto-detects everything (AMI, SSH keys, network)
- ✅ Ephemeral by default - no leftover resources
- ✅ Self-monitoring with **spored agent**
- ✅ Auto-terminates (TTL, idle timeout)
- ✅ Laptop-independent (spored runs ON the instance)

## 🚀 Quick Start

### Three Ways to Use spawn

```bash
# 1. Interactive Wizard (Beginner-Friendly)
spawn
# Guided setup, step-by-step
# Perfect for first-time users

# 2. From truffle (Power Users)
truffle search m7i.large | spawn

# 3. Direct with flags (Quick)
spawn --instance-type m7i.large --region us-east-1 --ttl 8h
```

## ✨ Features

### Core Features
- **🧙 Interactive Wizard**: Beginner-friendly guided setup
- **🤖 Smart Defaults**: Auto-detects AMI, architecture, GPU
- **🪟 Cross-Platform**: Windows 11, Linux, macOS
- **🔑 SSH Made Easy**: Auto-detects/creates `~/.ssh/id_rsa` (or `C:\Users\...\.ssh\id_rsa`)
- **🏗️ Auto-Infrastructure**: Creates VPC/subnet/security groups
- **💤 Hibernation**: Save costs on intermittent workloads
- **⏱️ TTL Support**: Auto-terminate after time limit
- **🕐 Idle Detection**: Auto-terminate/hibernate when idle
- **📊 Live Progress**: Real-time step-by-step updates
- **💰 Cost Estimates**: Shows hourly and total costs
- **🪣 S3 Distribution**: Fast regional downloads (~20ms)

### Advanced Features
- **🎮 GPU Support**: Auto-selects GPU-enabled AL2023 AMI
- **💪 Multi-Architecture**: x86_64 and ARM (Graviton)
- **🔄 Spot Instances**: Up to 70% savings
- **📡 spored Agent**: Self-monitoring (systemd service)
- **🔧 Laptop-Independent**: Works even when laptop is off

## 📦 Installation

```bash
# Clone repo
git clone https://github.com/yourusername/spawn
cd spawn

# Build
make build

# Install
sudo make install

# Verify
spawn version
spored version
```

### Pre-built Binaries

```bash
# Linux x86_64
curl -LO https://github.com/yourusername/spawn/releases/latest/download/spawn-linux-amd64
chmod +x spawn-linux-amd64
sudo mv spawn-linux-amd64 /usr/local/bin/spawn

# Linux ARM64 (Graviton)
curl -LO https://github.com/yourusername/spawn/releases/latest/download/spawn-linux-arm64
chmod +x spawn-linux-arm64
sudo mv spawn-linux-arm64 /usr/local/bin/spawn
```

## 🔑 AWS Prerequisites

### Required AWS Permissions

spawn requires specific IAM permissions to launch instances and manage resources. See [IAM_PERMISSIONS.md](IAM_PERMISSIONS.md) for the complete policy.

**Quick Setup:**

1. Validate your current permissions:
   ```bash
   ./scripts/validate-permissions.sh aws  # Replace 'aws' with your profile name
   ```

2. If missing permissions, attach the spawn policy to your IAM user:
   ```bash
   aws iam put-user-policy \
     --user-name your-username \
     --policy-name spawn-policy \
     --policy-document file://spawn/IAM_PERMISSIONS.md
   ```

**What spawn needs:**
- **EC2**: Launch instances, manage SSH keys, query instance types
- **IAM**: Create `spored-instance-role` (auto-created once per account)
- **SSM**: Query latest Amazon Linux 2023 AMI IDs

The IAM role (`spored-instance-role`) is automatically created the first time you launch an instance. This role allows the spored agent to:
- Read its own EC2 tags (for TTL/idle configuration)
- Terminate itself when TTL/idle limits are reached

**Security Note:** The spored role can only terminate instances tagged with `spawn:managed=true`.

For detailed information, see [IAM_PERMISSIONS.md](IAM_PERMISSIONS.md).

## 🎓 Usage

### Basic Launch

```bash
# From truffle (recommended)
truffle search m7i.large | spawn launch

# Direct
spawn launch --instance-type m7i.large --region us-east-1
```

### With Auto-Termination

```bash
# TTL (time limit)
spawn launch --ttl 24h

# Idle timeout
spawn launch --idle-timeout 1h

# Hibernate instead of terminate
spawn launch --idle-timeout 1h --hibernate-on-idle
```

### Spot Instances

```bash
# From truffle spot
truffle spot m7i.large --max-price 0.10 | spawn launch --spot

# Direct
spawn launch --instance-type m7i.large --spot --spot-max-price 0.10
```

### GPU Instances

```bash
# spawn auto-detects GPU and uses GPU-enabled AMI
truffle capacity --instance-types p5.48xlarge --available-only | \
  spawn launch --ttl 24h

# Auto-selects: ami-xxx (AL2023 with NVIDIA drivers)
```

### Hibernation

```bash
# Enable hibernation support
spawn launch --hibernate --ttl 24h

# Hibernate on idle
spawn launch --hibernate --idle-timeout 1h --hibernate-on-idle
```

## 🏗️ Architecture

### The spawn Ecosystem

```
┌─────────────────────────────────────────────────────────┐
│ Your Laptop                                             │
│                                                         │
│  truffle → finds instances                              │
│     ↓ (JSON via pipe)                                   │
│  spawn → launches instance                              │
│     ↓ (injects spored via user-data)                    │
└─────────────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────────────┐
│ EC2 Instance                                            │
│                                                         │
│  spored → systemd service                               │
│     • Reads tags (spawn:ttl, spawn:idle-timeout)        │
│     • Monitors CPU, network                             │
│     • Self-terminates when conditions met               │
│     • Hibernates if configured                          │
│                                                         │
│  💡 Laptop can close - spored handles everything!       │
└─────────────────────────────────────────────────────────┘
                    ↓ (on termination)
┌─────────────────────────────────────────────────────────┐
│ Cleanup (automatic)                                     │
│     • Finds resources with spawn:parent=i-xxx           │
│     • Deletes SGs, subnets, VPCs, keys                  │
│     • No orphaned resources!                            │
└─────────────────────────────────────────────────────────┘
```

### spored Agent

**spored** runs as a systemd service on each instance:

```bash
# On instance
systemctl status spored
journalctl -u spored -f

# Configuration via tags (set by spawn)
spawn:ttl=24h              # Auto-terminate after 24h
spawn:idle-timeout=1h       # Terminate if idle for 1h
spawn:hibernate-on-idle=true # Hibernate instead of terminate
spawn:idle-cpu=5            # CPU threshold for idle (default: 5%)
```

**What spored monitors:**
- ✅ Uptime vs TTL
- ✅ CPU usage (idle detection)
- ✅ Network traffic (idle detection)
- ✅ Warns users before termination (wall, /tmp/SPAWN_WARNING)

**Actions:**
- Self-terminates when TTL expires
- Self-terminates when idle timeout reached
- Self-hibernates if configured
- Logs to `/var/log/spored.log` and journald

## 🎨 AMI Selection

### Automatic (Recommended)

```bash
# spawn auto-detects the right AMI:
spawn launch --instance-type m7i.large
# → AL2023 standard x86_64

spawn launch --instance-type m8g.xlarge
# → AL2023 standard ARM64 (Graviton)

spawn launch --instance-type p5.48xlarge
# → AL2023 GPU-enabled x86_64 (NVIDIA drivers pre-installed)

spawn launch --instance-type g5g.xlarge
# → AL2023 GPU-enabled ARM64
```

### Manual Override

```bash
# Specific AMI
spawn launch --ami ami-1234567890abcdef0

# Future: AMI aliases
spawn launch --ami ubuntu
spawn launch --ami pytorch
```

### How It Works

spawn queries **AWS Systems Manager Parameter Store** for the latest official AMIs:

| Instance Type | Architecture | GPU | AMI Parameter |
|--------------|-------------|-----|---------------|
| m7i.* | x86_64 | No | `/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64` |
| m8g.* | arm64 | No | `/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64` |
| p5.*, g6.* | x86_64 | Yes | `/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-gpu-x86_64` |
| g5g.* | arm64 | Yes | `/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-gpu-arm64` |

Always gets the **latest** AMI automatically!

## 🔑 SSH Key Management

### Default Behavior

```bash
spawn launch
# 1. Checks for ~/.ssh/id_rsa.pub
# 2. If exists → uses it ✅
# 3. If not → offers to create
# 4. Uploads to AWS as "default-ssh-key" (if not already there)
# 5. Reuses on subsequent launches
```

### Custom Key

```bash
spawn launch --key-pair my-existing-key
```

### What spawn Does

- ✅ Uses your **existing** SSH key (`~/.ssh/id_rsa`)
- ✅ One key for all instances (standard Unix behavior)
- ✅ Uploads to AWS once, reuses forever
- ✅ Tags uploaded key: `spawn:imported=true, spawn:keep=true`
- ✅ Never deletes imported keys (they're yours!)

## 💰 Cost Awareness

### States and Costs

```
RUNNING 🟢
├─ Compute: $2.42/day
├─ Storage: $0.03/day
└─ Total: $2.45/day

STOPPED/HIBERNATED 🟡
├─ Compute: $0.00 (NOT RUNNING)
├─ Storage: $0.03/day (⚠️ STILL PAYING!)
└─ Total: $0.03/day

TERMINATED 🔴
└─ Everything: $0.00 (ALL CLEANED UP)
```

**Key Point:** Stopped ≠ Free! You still pay for storage.

## 🏷️ Resource Tagging

Every resource created by spawn is tagged:

```bash
# Instance (root)
spawn:managed=true          # Created by spawn
spawn:root=true             # Root resource (others are children)
spawn:ttl=24h               # Auto-terminate after 24h
spawn:idle-timeout=1h       # Terminate if idle
spawn:hibernate-on-idle=true

# Child resources (VPC, subnet, SG, key)
spawn:managed=true
spawn:parent=i-1234567890   # Parent instance ID
spawn:created-by=spawn
```

**Cleanup Logic:**
1. Instance terminates (TTL, idle, or manual)
2. Find all resources with `spawn:parent=i-xxx`
3. Delete in dependency order
4. No orphaned resources!

## 🎯 Real-World Examples

### Quick Dev Instance

```bash
truffle search t3.medium | spawn launch --ttl 8h --name dev-box
# Auto-terminates after work day
```

### GPU Training Job

```bash
truffle capacity --instance-types p5.48xlarge --available-only | \
  spawn launch \
    --use-reservation \
    --ttl 24h \
    --idle-timeout 2h \
    --user-data @train.sh \
    --name llm-training

# • Uses capacity reservation (guaranteed capacity)
# • Runs training script
# • Terminates if idle for 2h
# • OR after 24h maximum
# • Cleans up everything
# • Works even if laptop closes!
```

### Cost-Optimized Spot

```bash
truffle spot "m8g.*" --max-price 0.15 --sort-by-price | \
  spawn launch --spot --hibernate --idle-timeout 1h

# • Cheapest Graviton Spot instance
# • Hibernates when idle (saves 99% cost)
# • Resume quickly when needed
```

## 🛠️ Development

### Building

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Outputs:
# bin/spawn-linux-amd64
# bin/spawn-linux-arm64
# bin/spored-linux-amd64
# bin/spored-linux-arm64
# (+ macOS variants)
```

### Testing

```bash
make test
```

### Installing Locally

```bash
make install
# Installs to /usr/local/bin/
```

## 📊 spored Monitoring

### On Instance

```bash
# Check status
systemctl status spored

# View logs
journalctl -u spored -f

# Check configuration
cat /var/log/spored.log

# See warnings
cat /tmp/SPAWN_WARNING
```

### Metrics Monitored

- **Uptime**: vs TTL limit
- **CPU Usage**: Average over last 5 minutes
- **Network Traffic**: Bytes sent/received
- **Idle Time**: Time since last activity

### Idle Detection

Considered **idle** when:
- CPU < 5% (configurable via `spawn:idle-cpu` tag)
- Network < 10KB/min
- For duration > idle timeout

## 🔄 Integration with truffle

spawn is designed to work seamlessly with truffle:

```bash
# Instance discovery → Launch
truffle search m7i.large --pick-first --output json | spawn launch

# Spot pricing → Cheapest
truffle spot m7i.large --sort-by-price --output json | spawn launch --spot

# Capacity → Guaranteed
truffle capacity --gpu-only --available-only --output json | spawn launch

# Multi-AZ → HA
truffle az m7i.large --min-az-count 3 --output json | spawn launch
```

**JSON Flow:**
```json
{
  "instance_type": "m7i.large",
  "region": "us-east-1",
  "availability_zone": "us-east-1a",
  "architecture": "x86_64",
  "spot": true,
  "spot_price": 0.0331
}
```

spawn reads this and launches accordingly!

## ⚠️ Important Notes

### Hibernation Requirements

- ✅ Supported families: c5, m5, r5, c6i, m6i, r6i, m7i, m8g
- ✅ Requires encrypted EBS volume
- ✅ Volume must be large enough for RAM
- ✅ Not available on all instance types

### Spot Instances

- ⚠️ Can be interrupted (spawn handles gracefully)
- ✅ spored saves state before interruption
- ✅ Use with hibernation for best results

### GPU Instances

- ✅ Auto-selects GPU-enabled AMI (NVIDIA drivers pre-installed)
- ✅ Works with Capacity Blocks and ODCRs
- ⚠️ Very expensive - use TTL!

## 🎉 Summary

**spawn makes AWS simple:**

1. ✅ **Smart**: Auto-detects AMI, architecture, GPU
2. ✅ **Safe**: Auto-terminates, auto-cleans up
3. ✅ **Simple**: One command to launch
4. ✅ **Laptop-independent**: spored monitors from instance
5. ✅ **Cost-aware**: Hibernation, idle detection
6. ✅ **Non-expert friendly**: Just works!

**Perfect for:**
- Data scientists who need GPUs
- Developers who need test instances
- Anyone who wants compute NOW without AWS complexity

---

**Next Steps:**
- Try: `truffle search m7i.large | spawn launch`
- Read: `spawn launch --help`
- Monitor: `ssh instance; systemctl status spored`

**Companion Tools:**
- [truffle](../truffle) - Find the right instance type
- [spawn](.) - Launch it effortlessly
