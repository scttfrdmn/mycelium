# truffle + spawn Quick Reference

## 🚀 Common Commands

### truffle (No AWS creds needed for most)

```bash
# Find instance types
truffle search m7i.large
truffle search "m7i.*" --regions us-east-1

# Check Spot prices
truffle spot m7i.large
truffle spot "m7i.*" --sort-by-price

# Find ML capacity
truffle capacity --gpu-only
truffle capacity --instance-types p5.48xlarge

# With quotas (needs AWS creds)
truffle search m7i.large --check-quotas
truffle quotas
truffle quotas --family P --request
```

### spawn (Needs AWS creds)

```bash
# Interactive wizard (easiest!)
spawn

# From truffle (recommended)
truffle search m7i.large | spawn
truffle capacity --gpu-only | spawn --ttl 24h

# Direct launch
spawn --instance-type m7i.large --region us-east-1 --ttl 8h
```

---

## 🎯 Quick Workflows

### First Time Ever

```bash
# 1. No AWS account? Learn anyway!
truffle search m7i.large
# See specs, pricing, no account needed

# 2. Got AWS account? Configure it
aws configure

# 3. Check your quotas
truffle quotas

# 4. Launch your first instance
spawn
# Press Enter 6 times → done!
```

### Quick Dev Box

```bash
# Wizard
spawn
# Choose t3.medium, 8h TTL, auto-terminate

# Or one-liner
spawn --instance-type t3.medium --ttl 8h --idle-timeout 1h
```

### GPU Training

```bash
# 1. Check GPU quota (probably 0)
truffle quotas --family P

# 2. Request increase (if needed)
truffle quotas --family P --request

# 3. Find available capacity
truffle capacity --instance-types p5.48xlarge --check-quotas

# 4. Launch with auto-terminate
truffle capacity --instance-types p5.48xlarge | \
  spawn --ttl 24h --hibernate-on-idle
```

### Cheapest Spot

```bash
truffle spot "m7i.*" --sort-by-price --pick-first | spawn --spot
```

---

## 💡 Pro Tips

### Save Money
```bash
# Use Spot (70% cheaper)
spawn --spot

# Auto-terminate
spawn --ttl 8h --idle-timeout 1h

# Hibernate (saves 99% when idle)
spawn --hibernate --idle-timeout 30m
```

### Avoid Quota Failures
```bash
# Always check before launching
truffle search <type> --check-quotas | spawn
```

### Multi-Region
```bash
# Find region with best availability
truffle quotas --regions us-east-1,us-west-2,eu-west-1

# Launch in best region
truffle search m7i.large --regions us-west-2 | spawn
```

### Windows Users
```powershell
# Everything works on Windows!
PS C:\> spawn
PS C:\> truffle quotas
# Uses C:\Users\username\.ssh\id_rsa
```

---

## 🔑 AWS Credentials

### Required For

- `spawn` (all commands)
- `truffle quotas`
- `truffle ... --check-quotas`

### NOT Required For

- `truffle search`
- `truffle spot`
- `truffle capacity`

### Setup

```bash
# Option 1: Environment variables
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
export AWS_DEFAULT_REGION=us-east-1

# Option 2: AWS CLI
aws configure

# Option 3: IAM roles (on EC2)
# Automatic!
```

---

## ⚠️ Common Issues

### "VcpuLimitExceeded"
```bash
# Check quotas first!
truffle quotas
truffle quotas --family P --request
```

### "No SSH key found"
```bash
# Use wizard (creates key automatically)
spawn

# Or create manually
ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa
```

### "Cannot check quotas"
```bash
# Configure AWS credentials
aws configure
```

### "spawnd not found on instance"
```bash
# Check S3 buckets deployed
./scripts/deploy-spawnd.sh 0.1.0
```

---

## 📊 Flags Reference

### truffle search

```
--regions              Regions to search (default: us-east-1)
--check-quotas         Check AWS quotas (needs creds)
--show-all             Show even quota-blocked instances
--output json          JSON output (for piping)
--pick-first           Return first result only
```

### truffle quotas

```
--regions              Regions to check (default: us-east-1)
--family               Filter by family (Standard, P, G, etc.)
--request              Generate quota increase commands
```

### spawn

```
--instance-type        Instance type (or from stdin)
--region               AWS region
--spot                 Use Spot instances
--ttl                  Auto-terminate after duration (e.g., 8h)
--idle-timeout         Auto-terminate if idle (e.g., 1h)
--hibernate            Enable hibernation
--hibernate-on-idle    Hibernate instead of terminate
--interactive          Force wizard mode
```

---

## 🎨 Output Formats

### truffle JSON (for piping)

```bash
truffle search m7i.large --output json
```

```json
{
  "instance_type": "m7i.large",
  "region": "us-east-1",
  "vcpus": 2,
  "memory_mib": 8192,
  "quota_can_launch": true
}
```

### spawn

```
╔════════════════════════════════════════════════════════╗
║  🎉 Instance Ready!                                    ║
╚════════════════════════════════════════════════════════╝

Instance ID:  i-1234567890abcdef0
Public IP:    54.123.45.67

🔌 Connect Now:
  ssh -i ~/.ssh/id_rsa ec2-user@54.123.45.67

💡 Will auto-terminate after 8h
```

---

## 🆘 Help Commands

```bash
# General help
truffle --help
spawn --help

# Command-specific help
truffle search --help
truffle quotas --help
spawn launch --help

# Version
truffle --version
spawn --version
```

---

## 📚 Documentation

- **truffle README**: Full search/spot/capacity docs
- **truffle QUOTAS.md**: Quota checking guide
- **spawn README**: Full launch/wizard docs
- **spawn ENHANCEMENTS.md**: S3/Windows/Wizard details
- **COMPLETE_ECOSYSTEM.md**: Everything together

---

## 🎯 Decision Tree

```
Need AWS instance?
  ├─ First time?
  │  └─ spawn (wizard mode)
  │
  ├─ Know what you want?
  │  └─ spawn --instance-type ... --ttl 8h
  │
  ├─ Want cheapest?
  │  └─ truffle spot ... | spawn --spot
  │
  ├─ Need GPU?
  │  ├─ truffle quotas --family P
  │  └─ truffle capacity --gpu-only | spawn --ttl 24h
  │
  └─ Power user?
     └─ truffle search ... --check-quotas | spawn
```

---

## ✅ Best Practices

1. **Always use --check-quotas** for production
2. **Always set --ttl** to prevent surprise bills
3. **Use --hibernate** for intermittent workloads
4. **Request GPU quotas early** (takes 24h)
5. **Check quotas in all regions** for best availability
6. **Use Spot for dev/test** (70% cheaper)
7. **Let spawnd monitor** (close laptop safely)

---

**Remember:** 
- truffle = Find
- spawn = Launch
- spawnd = Monitor (automatic)

**You're ready to use AWS like a pro!** 🚀
