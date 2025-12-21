# mycelium

> The underground network for AWS compute

**mycelium** is a suite of tools that makes AWS EC2 accessible to everyone - from complete beginners to ML engineers.

```
┌─────────────────────────────────────────────────────────┐
│  🔍 truffle  - Find instances, check quotas            │
│  🚀 spawn    - Launch effortlessly                      │
│  🤖 spawnd   - Monitor automatically                    │
└─────────────────────────────────────────────────────────┘

Making AWS accessible to everyone.
```

---

## 🚀 Quick Start

### Installation

```bash
# Extract the archive (you already did this!)
cd mycelium

# Build all tools
make build-all

# Or build individually
cd truffle && make build
cd spawn && make build
```

### First Launch (2 minutes!)

```bash
# Interactive wizard - perfect for beginners
./spawn/bin/spawn

# Press Enter 6 times → Instance ready!
```

### Power User Flow

```bash
# Find cheapest Spot instance
./truffle/bin/truffle spot "m7i.*" --sort-by-price --pick-first | \
  ./spawn/bin/spawn --ttl 8h
```

---

## 🍄 The Tools

### truffle - Find & Discover

Search instance types, check Spot prices, find GPU capacity, manage quotas.

**Works without AWS account!** (credentials optional for quota checking)

```bash
truffle search m7i.large           # Search instances
truffle spot m7i.large             # Check Spot prices
truffle capacity --gpu-only        # Find GPU capacity
truffle quotas                     # Check quotas (needs AWS creds)
```

[Full documentation →](truffle/README.md)

### spawn - Launch Effortlessly

Launch instances with wizard, direct commands, or pipe from truffle.

**Requires AWS credentials.**

```bash
spawn                              # Wizard mode
spawn --instance-type m7i.large    # Direct launch
truffle search ... | spawn         # Pipe mode
```

[Full documentation →](spawn/README.md)

### spawnd - Monitor Automatically

Runs on your instance as a systemd service. Monitors TTL and idle time, auto-terminates or hibernates.

**No user interaction needed** - spawnd reads its configuration from AWS tags.

---

## 🎯 Who Is This For?

- **Beginners**: Interactive wizard, no AWS knowledge needed
- **Data Scientists**: GPU access without DevOps, quota management
- **Developers**: Quick dev boxes, cost-effective Spot instances
- **ML Engineers**: Capacity discovery, hibernation, auto-termination
- **Windows Users**: Native support (finally!)

---

## 📚 Documentation

### User Guides
- **QUICK_REFERENCE.md** - Command cheat sheet
- **COMPLETE_ECOSYSTEM.md** - Full overview
- **truffle/README.md** - truffle user guide
- **truffle/QUOTAS.md** - Quota checking guide
- **spawn/README.md** - spawn user guide
- **spawn/ENHANCEMENTS.md** - S3/Windows/Wizard details

### Deployment & Security (For Organizations)
- **DEPLOYMENT_GUIDE.md** - Enterprise deployment strategies
- **SECURITY.md** - Comprehensive security documentation for CISOs
- **spawn/IAM_PERMISSIONS.md** - Required IAM permissions
- **scripts/setup-spawnd-iam-role.sh** - One-time IAM role setup script
- **scripts/validate-permissions.sh** - Permission validation tool

---

## 🔧 Building

### Prerequisites

- Go 1.21+
- AWS account (for spawn)
- AWS credentials (optional for truffle)

### Build Commands

```bash
# Build everything (current platform)
make build-all

# Build for all platforms
cd truffle && make build-all
cd spawn && make build-all

# Install locally
cd truffle && sudo make install
cd spawn && sudo make install
```

---

## 🔑 AWS Credentials

### For spawn (Required)

```bash
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
export AWS_DEFAULT_REGION=us-east-1

# Or use: aws configure
```

### For truffle (Optional)

Most truffle features work **without credentials**:
- `truffle search` ✅
- `truffle spot` ✅
- `truffle capacity` ✅

Credentials only needed for:
- `truffle quotas`
- `truffle search --check-quotas`

---

## 🎨 Examples

### Absolute Beginner (First Time)

```bash
$ spawn

🧙 spawn Setup Wizard
[Press Enter 6 times with defaults]

🎉 Instance ready in 60 seconds!
ssh -i ~/.ssh/id_rsa ec2-user@54.123.45.67
```

### GPU Training

```bash
# Check GPU quota
$ truffle quotas --family P
🔴 P: 0/0 vCPUs (zero quota)

# Request increase
$ truffle quotas --family P --request
[Copy/paste AWS command, wait 24h]

# Launch GPU instance
$ truffle capacity --instance-types p5.48xlarge --check-quotas | \
    spawn --ttl 24h --hibernate-on-idle
```

### Cheapest Dev Box

```bash
$ truffle spot "t3.*" --sort-by-price --pick-first | \
    spawn --spot --ttl 8h

# Cost: ~$0.01/hr
# Auto-terminates after 8h
```

---

## 🌟 Key Features

- ✅ **Zero to instance in 2 minutes**
- ✅ **No surprise bills** (auto-termination)
- ✅ **Works on Windows** (native support)
- ✅ **Quota-aware** (prevents failures)
- ✅ **GPU support** (auto-detects AMI)
- ✅ **Hibernation** (save 99% when idle)
- ✅ **Cross-platform** (Windows/Linux/macOS)
- ✅ **Production-ready** (error handling, logging)

---

## 📦 Project Structure

```
mycelium/
├── README.md                    ← You are here
├── QUICK_REFERENCE.md          
├── COMPLETE_ECOSYSTEM.md       
├── Makefile                    
│
├── truffle/                     ← Find instances
│   ├── README.md
│   ├── QUOTAS.md
│   ├── cmd/
│   ├── pkg/
│   └── bindings/
│
└── spawn/                       ← Launch instances
    ├── README.md
    ├── ENHANCEMENTS.md
    ├── cmd/
    ├── pkg/
    └── scripts/
```

---

## 💬 Quick Commands

```bash
# Discovery (no AWS account needed)
truffle search m7i.large
truffle spot m7i.large

# With AWS credentials
truffle quotas
spawn                              # Wizard
spawn --instance-type m7i.large    # Direct

# Power user
truffle search ... --check-quotas | spawn
```

---

**Making AWS accessible to everyone, one instance at a time.** 🍄✨

**Ready to grow your cloud infrastructure naturally!** 🌱
