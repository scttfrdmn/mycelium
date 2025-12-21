# 🚀 Quick Start Guide

Get started with Truffle in under 5 minutes!

## 1. Installation

### Option A: Build from source
```bash
git clone https://github.com/yourusername/truffle.git
cd truffle
go mod download
go build -o truffle
```

### Option B: Download binary
```bash
# Linux
curl -LO https://github.com/yourusername/truffle/releases/latest/download/truffle-linux-amd64
chmod +x truffle-linux-amd64
sudo mv truffle-linux-amd64 /usr/local/bin/truffle

# macOS (Apple Silicon)
curl -LO https://github.com/yourusername/truffle/releases/latest/download/truffle-darwin-arm64
chmod +x truffle-darwin-arm64
sudo mv truffle-darwin-arm64 /usr/local/bin/truffle
```

## 2. Configure AWS Credentials

**Easiest way (recommended):**
```bash
# Modern AWS login - handles everything!
aws login
```

**That's it!** ✨ The `aws login` command:
- Authenticates you with AWS
- Handles MFA automatically  
- Works with SSO/IAM Identity Center
- Refreshes credentials automatically

**For SSO users:**
```bash
aws login --profile my-sso-profile
export AWS_PROFILE=my-sso-profile
```

**Alternative methods (if not using aws login):**
```bash
# Option 1: Environment variables
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key

# Option 2: AWS CLI configure
aws configure
```

## 3. Your First Search

```bash
# Search for m7i.large in all regions (7th gen Intel)
truffle search m7i.large
```

Output:
```
🍄 Found 1 instance type(s) across 17 region(s)

+---------------+--------------+-------+--------------+--------------+
| Instance Type | Region       | vCPUs | Memory (GiB) | Architecture |
+---------------+--------------+-------+--------------+--------------+
| m7i.large     | us-east-1    |     2 |          8.0 | x86_64      |
|               | us-east-2    |     2 |          8.0 | x86_64      |
|               | us-west-1    |     2 |          8.0 | x86_64      |
|               | us-west-2    |     2 |          8.0 | x86_64      |
...
+---------------+--------------+-------+--------------+--------------+
```

## 4. Common Use Cases

### Search with wildcards
```bash
# Find all m7i family instances (latest Intel)
truffle search "m7i.*"

# Find all Graviton4 xlarge instances
truffle search "m8g.*.xlarge"
```

### Filter by region
```bash
# Check specific regions
truffle search m7i.large --regions us-east-1,eu-west-1
```

### Get JSON output
```bash
# For scripting and automation
truffle search m7i.large --output json
```

### Include availability zones
```bash
# See which AZs support the instance
truffle search m7i.large --include-azs
```

### Filter by specifications
```bash
# Find Graviton4 instances with at least 4 vCPUs and 16 GiB RAM
truffle search "m8g.*" --min-vcpu 4 --min-memory 16
```

## 5. List Available Types

```bash
# List all instance families
truffle list --family

# List all instance sizes
truffle list --sizes
```

## 6. Advanced Examples

### Multi-region deployment check
```bash
# Create a script to validate instance availability
cat > check-deployment.sh << 'EOF'
#!/bin/bash
INSTANCE="m7i.medium"
REGIONS=("us-east-1" "us-west-2" "eu-west-1")

for region in "${REGIONS[@]}"; do
    echo -n "Checking $region... "
    if truffle search "$INSTANCE" --regions "$region" --output json | jq -e '. | length > 0' > /dev/null; then
        echo "✅ Available"
    else
        echo "❌ Not available"
    fi
done
EOF

chmod +x check-deployment.sh
./check-deployment.sh
```

### Export to CSV for analysis
```bash
# Export all c7i instances to CSV (7th gen Intel)
truffle search "c7i.*" --output csv > c7i-instances.csv
```

### Terraform integration
```bash
# Generate list of regions for Terraform
truffle search m7i.xlarge --output json | \
    jq -r '[.[].region] | unique | .[]' | \
    awk '{printf "\"%s\",\n", $0}'
```

## 7. Tips & Tricks

### Use verbose mode for debugging
```bash
truffle search m7i.large --verbose
```

### Disable colors for piping
```bash
truffle search m7i.large --no-color | less
```

### Set default output format
```bash
# Add to your shell profile
alias truffle='truffle --output json'
```

### Combine with jq for filtering
```bash
# Get only us-east regions
truffle search m7i.large --output json | \
    jq '.[] | select(.region | startswith("us-east"))'
```

## 8. Getting Help

```bash
# General help
truffle --help

# Command-specific help
truffle search --help
truffle list --help

# Version information
truffle version
```

## 9. Next Steps

- 📖 Read the full [README.md](README.md) for detailed documentation
- 🔧 Check out [examples/](examples/) for more use cases
- 🐛 [Report issues](https://github.com/yourusername/truffle/issues)
- 💡 [Request features](https://github.com/yourusername/truffle/issues/new)

## Troubleshooting

### "AWS credentials not found"
```bash
# Easiest fix - just login!
aws login

# Verify it worked
aws sts get-caller-identity

# If using a profile
aws login --profile my-profile
export AWS_PROFILE=my-profile
```

### "Permission denied"
Make sure your AWS credentials have EC2 read permissions:
- `ec2:DescribeRegions`
- `ec2:DescribeInstanceTypes`
- `ec2:DescribeInstanceTypeOfferings`

### Slow searches
Use region filtering to speed up searches:
```bash
# Only search specific regions
truffle search m5.large --regions us-east-1,us-west-2
```

---

🎉 **You're all set!** Start exploring AWS instance types with Truffle!
