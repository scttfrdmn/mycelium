<p align="center">
  <img src="../assets/logo-light.png#gh-light-mode-only" alt="Mycelium Logo" width="500">
  <img src="../assets/logo-dark.png#gh-dark-mode-only" alt="Mycelium Logo" width="500">
</p>

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
- **✅ Completion Signals**: Workloads signal when done, trigger auto-cleanup
- **📊 Live Progress**: Real-time step-by-step updates
- **💰 Cost Estimates**: Shows hourly and total costs
- **🪣 S3 Distribution**: Fast regional downloads (~20ms)
- **🌍 Multilingual**: 6 languages supported (en, es, fr, de, ja, pt)

### Advanced Features
- **🎮 GPU Support**: Auto-selects GPU-enabled AL2023 AMI
- **💪 Multi-Architecture**: x86_64 and ARM (Graviton)
- **🔄 Spot Instances**: Up to 70% savings
- **📡 spored Agent**: Self-monitoring (systemd service)
- **🔧 Laptop-Independent**: Works even when laptop is off
- **♿ Accessibility**: Screen reader support with --accessibility flag
- **🔢 Job Arrays**: Launch coordinated instance groups for MPI, distributed training, parameter sweeps ([docs](JOB_ARRAYS.md))
- **💾 AMI Management**: Create and manage custom AMIs for reusable software stacks ([docs](AMI_MANAGEMENT.md))

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

## 🌍 Internationalization

spawn supports multiple languages for a better user experience worldwide.

### Supported Languages

- 🇬🇧 **English** (en) - Default
- 🇪🇸 **Spanish** (es) - Español
- 🇫🇷 **French** (fr) - Français
- 🇩🇪 **German** (de) - Deutsch
- 🇯🇵 **Japanese** (ja) - 日本語
- 🇵🇹 **Portuguese** (pt) - Português

### Using Different Languages

```bash
# Spanish
spawn --lang es
spawn --lang es launch --instance-type m7i.large

# Japanese
spawn --lang ja
spawn --lang ja launch

# French
spawn --lang fr --help
spawn --lang fr list

# German
spawn --lang de connect i-1234567890

# Portuguese
spawn --lang pt launch --interactive
```

### Environment Variable

Set your preferred language globally:

```bash
# Set language in your shell profile (~/.bashrc, ~/.zshrc)
export SPAWN_LANG=es

# Now all spawn commands use Spanish
spawn launch
spawn list
spawn --help
```

### Language Detection Priority

spawn detects language in this order:

1. `--lang` flag (highest priority)
2. `SPAWN_LANG` environment variable
3. Config file (`~/.spawn/config.yaml`)
4. System locale (`LANG`, `LC_ALL`)
5. Default to English

### What Gets Translated

All user-facing text is translated:

- ✅ Command descriptions and help text
- ✅ Interactive wizard (all 6 steps)
- ✅ Progress indicators and status messages
- ✅ Success/warning/error messages
- ✅ Table headers and output
- ✅ Flag descriptions

**Technical terms stay in English** (AWS, EC2, AMI, VPC, SSH) for consistency with AWS documentation.

### Accessibility Features

Screen reader-friendly output:

```bash
# Disable emoji only
spawn --no-emoji launch

# Full accessibility mode (no emoji, no color, screen reader-friendly)
spawn --accessibility launch
spawn --accessibility list
```

**Accessibility mode:**
- Replaces emoji with text symbols (`[✓]`, `[✗]`, `[!]`, `[*]`)
- Disables color output
- Uses clear status announcements
- Works with JAWS, NVDA, VoiceOver

### Examples in Different Languages

**Spanish Interactive Wizard:**
```bash
$ spawn --lang es

🧙 Asistente de Configuración de spawn

Paso 1 de 6: Elige el tipo de instancia
[Interactive Spanish wizard...]
```

**Japanese Launch:**
```bash
$ spawn --lang ja launch --instance-type m7i.large

インスタンスを生成中...
  ✓ AMIを検出中
  ✓ インスタンスを起動中
  ✓ IPアドレスを待機中
  ✓ SSHの準備完了を待機中

成功！インスタンスの準備が完了しました
```

**French List:**
```bash
$ spawn --lang fr list

Recherche d'instances gérées par spawn dans toutes les régions...

+---------------+-----------+--------+----------------+
| ID Instance   | Région    | État   | IP Publique    |
+---------------+-----------+--------+----------------+
| i-1234567890  | us-east-1 | actif  | 54.123.45.67   |
+---------------+-----------+--------+----------------+
```

## 📋 Commands

### spawn list

List all spawn-managed instances across regions with powerful filtering options.

```bash
# List all instances across all regions
spawn list

# Filter by specific region
spawn list --region us-east-1

# Filter by instance state
spawn list --state running
spawn list --state stopped

# Filter by instance type (exact match)
spawn list --instance-type m7i.large

# Filter by instance family (all sizes in family)
spawn list --family m7i
# Matches: m7i.large, m7i.xlarge, m7i.2xlarge, etc.

# Filter by tag
spawn list --tag env=prod
spawn list --tag Name=my-instance

# Combine multiple filters (AND logic)
spawn list --region us-east-1 --state running --family m7i --tag env=prod
# Only shows instances matching ALL criteria

# JSON output for automation
spawn list --format json
spawn list --format json | jq '.[] | select(.State == "running")'

# YAML output
spawn list --format yaml
```

**Output format:**

```
Finding spawn-managed instances in all regions...

+------------------+------------+---------+----------------+--------+-------+
| Instance ID      | Region     | State   | Public IP      | Type   | Age   |
+------------------+------------+---------+----------------+--------+-------+
| i-0123456789abc  | us-east-1  | running | 54.123.45.67   | m7i.lg | 2h30m |
| i-0987654321def  | us-west-2  | running | 52.98.76.54    | t3.med | 5d6h  |
+------------------+------------+---------+----------------+--------+-------+

Total: 2 instances
```

**What gets listed:**
- Only instances tagged with `spawn:managed=true`
- Searches all regions by default (unless `--region` specified)
- Shows: ID, region, state, public IP, instance type, age
- Age format: `2h30m` (2 hours 30 minutes), `5d6h` (5 days 6 hours)

### spawn extend

Extend the TTL (time-to-live) for running instances to prevent automatic termination.

```bash
# Extend by instance ID
spawn extend i-0123456789abcdef 2h
# Adds 2 hours to current TTL

# Extend by instance name
spawn extend my-instance 1d
# Adds 1 day to current TTL

# Various time formats supported
spawn extend i-xxx 30m      # 30 minutes
spawn extend i-xxx 2h       # 2 hours
spawn extend i-xxx 1d       # 1 day
spawn extend i-xxx 3h30m    # 3 hours 30 minutes
spawn extend i-xxx 2d12h    # 2 days 12 hours
spawn extend i-xxx 1d2h30m  # 1 day 2 hours 30 minutes

# For long-running tasks
spawn extend training-job 24h
# Prevents termination for next 24 hours
```

**How it works:**
1. Reads current `spawn:ttl` tag from instance
2. Parses duration (e.g., "2h", "30m", "1d")
3. Updates the tag with extended time
4. Updates the spored agent configuration (if instance has spored)
5. Confirms extension with formatted duration

**Example output:**

```
Extending TTL for instance i-0123456789abcdef...
  ✓ Current TTL: 4 hours
  ✓ Extension: 2 hours
  ✓ New TTL: 6 hours

Instance will now run for approximately 6 hours from now.
```

**TTL format rules:**
- Valid units: `s` (seconds), `m` (minutes), `h` (hours), `d` (days)
- Can combine multiple units: `2h30m`, `1d12h`, `3h45m15s`
- Order doesn't matter: `2h30m` and `30m2h` are both valid
- No spaces allowed: `2h30m` ✅, `2h 30m` ❌

**Requirements:**
- Instance must be running
- Instance must have `spawn:managed=true` tag
- Only works on instances created by spawn

### spawn connect / spawn ssh

Connect to instances via SSH. Both commands are **aliases** - they work identically.

```bash
# Connect by instance ID (both commands work the same)
spawn connect i-0123456789abcdef
spawn ssh i-0123456789abcdef

# Connect by instance name
spawn connect my-instance
spawn ssh my-instance

# Specify custom user
spawn connect i-xxx --user ubuntu
spawn ssh my-instance --user ec2-user

# Specify custom port
spawn connect i-xxx --port 2222
spawn ssh i-xxx --port 2222

# Force Session Manager (bypasses public IP)
spawn connect i-xxx --session-manager
spawn ssh i-xxx --session-manager

# Custom SSH key
spawn connect i-xxx --key ~/.ssh/my-key.pem
spawn ssh i-xxx --key my-key  # Auto-finds ~/.ssh/my-key or my-key.pem
```

**SSH key resolution:**

spawn searches for SSH keys in this order:
1. Exact name: `~/.ssh/my-key`
2. With .pem extension: `~/.ssh/my-key.pem`
3. With .key extension: `~/.ssh/my-key.key`
4. Default keys: `~/.ssh/id_rsa`, `~/.ssh/id_ed25519`, `~/.ssh/id_ecdsa`

**Connection methods:**

1. **Public IP + SSH** (default, fastest)
   - Requires instance to have public IP
   - Uses SSH key from `~/.ssh/`
   - Direct connection

2. **AWS Session Manager** (fallback or forced)
   - No public IP required
   - Uses AWS SSM (Systems Manager)
   - Requires `session-manager-plugin` installed
   - Works through AWS API

**Examples:**

```bash
# Quick connect to recent instance
spawn list | grep running | head -1
# Copy the instance ID
spawn ssh i-0123456789abcdef

# Connect with custom user for Ubuntu
spawn ssh ubuntu-instance --user ubuntu

# Connect to private instance (no public IP)
spawn connect private-instance --session-manager

# Connect with specific key
spawn connect i-xxx --key my-project-key
```

**What happens:**
1. Resolves instance ID (if name provided, looks up by Name tag)
2. Waits for instance to be running
3. Checks for public IP
4. Finds SSH key (searches `~/.ssh/` directory)
5. Establishes SSH connection
6. Falls back to Session Manager if public IP unavailable

**Default values:**
- User: Detected from AMI (usually `ec2-user` for Amazon Linux)
- Port: `22`
- Key: Auto-detected from `~/.ssh/` (checks multiple patterns)

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

### With Completion Signals

Allow workloads to signal completion and trigger automatic cleanup:

```bash
# Batch job - terminate when job signals completion
spawn launch --on-complete terminate --ttl 4h
# Run your job, then: spored complete

# CI runner - terminate on completion, max 30 minutes
spawn launch --on-complete terminate --ttl 30m
# Job finishes, signals completion, instance terminates

# ML training - stop (preserve state) on completion
spawn launch --on-complete stop --ttl 8h
# Training finishes, signals completion, instance stops (can resume later)

# Data pipeline - hibernate for cost savings
spawn launch --on-complete hibernate --ttl 12h
# Pipeline finishes, signals completion, instance hibernates
```

**Signal completion two ways:**

```bash
# 1. Using spored CLI (recommended)
spored complete
spored complete --status success
spored complete --status success --message "Job completed successfully"

# 2. From any language (universal)
touch /tmp/SPAWN_COMPLETE

# With optional metadata (JSON)
echo '{"status":"success","message":"Build completed"}' > /tmp/SPAWN_COMPLETE
```

**How it works:**
1. Launch instance with `--on-complete` flag (terminate, stop, or hibernate)
2. spored monitors completion file (`/tmp/SPAWN_COMPLETE` by default)
3. When file appears, spored waits grace period (30s default)
4. Action executes automatically (terminate/stop/hibernate)

**Priority order:**
1. Spot interruption (highest priority)
2. Completion signal
3. TTL expiration
4. Idle timeout

**Advanced options:**

```bash
# Custom completion file
spawn launch --on-complete terminate --completion-file /app/done

# Custom grace period
spawn launch --on-complete terminate --completion-delay 1m
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
spawn:ttl=24h                 # Auto-terminate after 24h
spawn:idle-timeout=1h         # Terminate if idle for 1h
spawn:hibernate-on-idle=true  # Hibernate instead of terminate
spawn:idle-cpu=5              # CPU threshold for idle (default: 5%)
spawn:on-complete=terminate   # Action on completion signal
spawn:completion-file=/tmp/SPAWN_COMPLETE  # File to watch
spawn:completion-delay=30s    # Grace period before action
```

**What spored monitors:**
- ✅ Spot interruption warnings (highest priority)
- ✅ Completion signals (file-based)
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

### Batch Processing with Completion Signal

```bash
# Launch instance for batch job
truffle search m7i.large | \
  spawn launch \
    --on-complete terminate \
    --ttl 4h \
    --name batch-processor

# SSH to instance
spawn connect batch-processor

# Run your batch job (example)
cat > process.sh <<'EOF'
#!/bin/bash
echo "Processing data..."
python3 process_data.py
echo "Uploading results..."
aws s3 cp results.csv s3://my-bucket/
echo "Job complete!"
# Signal completion when done
spored complete --status success --message "Processed 10000 records"
EOF

chmod +x process.sh
./process.sh

# Instance automatically terminates 30 seconds after spored complete
# TTL (4h) acts as safety net if job hangs
# All resources cleaned up automatically
```

### CI/CD Runner with Self-Cleanup

```bash
# Launch ephemeral CI runner
spawn launch \
  --instance-type t3.large \
  --on-complete terminate \
  --ttl 30m \
  --name ci-runner \
  --user-data @setup-ci.sh

# setup-ci.sh
cat > setup-ci.sh <<'EOF'
#!/bin/bash
# Pull code
git clone https://github.com/user/repo
cd repo

# Run tests
npm install
npm test

# Signal completion (success or failure)
if [ $? -eq 0 ]; then
  spored complete --status success --message "All tests passed"
else
  spored complete --status failed --message "Tests failed"
fi
# Instance terminates automatically after 30s
EOF

# Instance runs job, signals completion, terminates
# No manual cleanup needed
# Works even if your laptop is off
```

### Job Array - Parameter Sweep

```bash
# Launch 100 instances for hyperparameter tuning
spawn launch \
  --count 100 \
  --job-array-name hyperparam-sweep \
  --instance-type t3.small \
  --ttl 2h \
  --command "python train.py --param-id \$JOB_ARRAY_INDEX"

# Each instance:
# - Gets unique index (0-99)
# - Knows total size (100)
# - Runs independently (no coordination needed)
# - Auto-terminates after 2h

# Monitor progress
spawn list --job-array-name hyperparam-sweep

# Extend if needed
spawn extend --job-array-name hyperparam-sweep 1h
```

### Job Array - MPI Cluster

```bash
# Launch 8-node MPI cluster
spawn launch \
  --count 8 \
  --job-array-name mpi-job \
  --instance-type c7i.4xlarge \
  --ttl 4h \
  --user-data @mpi-setup.sh

# mpi-setup.sh:
cat > mpi-setup.sh <<'EOF'
#!/bin/bash
# Install MPI
sudo yum install -y openmpi openmpi-devel

# Wait for all peers (barrier)
while [ ! -f /etc/spawn/job-array-peers.json ]; do sleep 1; done

# Generate hostfile
jq -r '.[] | .dns' /etc/spawn/job-array-peers.json > /tmp/hostfile

# Run MPI job (leader only)
if [ "$JOB_ARRAY_INDEX" -eq 0 ]; then
    mpirun -np $JOB_ARRAY_SIZE -hostfile /tmp/hostfile ./my-mpi-app
fi
EOF

# All instances coordinate automatically
# Peer discovery built-in
# Group DNS: mpi-job.account.spore.host → all IPs
```

See [JOB_ARRAYS.md](JOB_ARRAYS.md) for complete documentation.

### Custom AMI - Reusable Software Stack

```bash
# Step 1: Launch instance and install software
spawn launch --instance-type g5.xlarge --name pytorch-builder --ttl 2h
spawn connect pytorch-builder

# Install PyTorch + dependencies
pip3 install torch torchvision torchaudio transformers accelerate
# ... more installation ...
exit

# Step 2: Create AMI
spawn create-ami pytorch-builder \
  --name pytorch-2.2-cuda12-$(date +%Y%m%d) \
  --tag stack=pytorch \
  --tag version=2.2 \
  --tag cuda-version=12.1

# Step 3: List AMIs
spawn list-amis --stack pytorch

# Step 4: Launch from custom AMI (instant, no installation wait!)
AMI=$(spawn list-amis --stack pytorch --json | jq -r '.[0].ami_id')
spawn launch --instance-type g5.xlarge --ami $AMI --name training-job

# Launch time: 30 seconds vs 15+ minutes with user-data!
```

See [AMI_MANAGEMENT.md](AMI_MANAGEMENT.md) for complete documentation.

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

# Signal completion (workload finished)
spored complete
spored complete --status success --message "Job completed"

# View help
spored help
spored complete --help
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

---

## 📚 Documentation

- **[JOB_ARRAYS.md](JOB_ARRAYS.md)** - Job arrays for coordinated instance groups
- **[AMI_MANAGEMENT.md](AMI_MANAGEMENT.md)** - Create and manage custom AMIs
- **[IAM_PERMISSIONS.md](IAM_PERMISSIONS.md)** - Required AWS permissions
- **[DNS_SETUP.md](DNS_SETUP.md)** - Custom DNS configuration
- **[MONITORING.md](MONITORING.md)** - Monitoring and observability
- **[SHELL_COMPLETION.md](SHELL_COMPLETION.md)** - Shell completion setup
- **[SECURITY.md](SECURITY.md)** - Security considerations
