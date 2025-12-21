# Installation Guide

## Prerequisites

1. **Go 1.22 or higher**
   ```bash
   go version
   ```

2. **AWS CLI configured** (optional but recommended)
   ```bash
   aws configure
   ```

3. **Git** (for cloning the repository)

## Installation Methods

### Method 1: From Source (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/truffle.git
cd truffle

# Download dependencies
go mod download

# Build the binary
go build -o truffle

# Test the build
./truffle --help

# Optional: Install globally
sudo mv truffle /usr/local/bin/
```

### Method 2: Using Make

```bash
# Clone and build
git clone https://github.com/yourusername/truffle.git
cd truffle

# Download dependencies and build
make deps
make build

# Install system-wide
sudo make install
```

### Method 3: Using Go Install

```bash
go install github.com/yourusername/truffle@latest
```

### Method 4: Download Pre-built Binary

Download the appropriate binary for your platform from the [Releases](https://github.com/yourusername/truffle/releases) page:

```bash
# Linux AMD64
curl -LO https://github.com/yourusername/truffle/releases/latest/download/truffle-linux-amd64
chmod +x truffle-linux-amd64
sudo mv truffle-linux-amd64 /usr/local/bin/truffle

# macOS ARM64 (Apple Silicon)
curl -LO https://github.com/yourusername/truffle/releases/latest/download/truffle-darwin-arm64
chmod +x truffle-darwin-arm64
sudo mv truffle-darwin-arm64 /usr/local/bin/truffle

# Windows
# Download truffle-windows-amd64.exe and add to PATH
```

## AWS Configuration

### Method 1: AWS Login (Recommended - Super Easy!) ⭐

The modern, easiest way to authenticate:

```bash
# Simple login
aws login

# That's it! You're authenticated! ✅
```

**For AWS SSO / IAM Identity Center:**
```bash
# Login with a specific profile
aws login --profile my-sso-profile

# Use that profile with Truffle
export AWS_PROFILE=my-sso-profile
truffle search m7i.large
```

**Benefits:**
- ✅ No manual credential copying
- ✅ Automatic credential refresh
- ✅ Works with MFA automatically
- ✅ Perfect for SSO/IAM Identity Center
- ✅ Handles temporary credentials seamlessly

### Method 2: AWS Credentials File (Traditional)

Create `~/.aws/credentials`:

```ini
[default]
aws_access_key_id = YOUR_ACCESS_KEY
aws_secret_access_key = YOUR_SECRET_KEY
```

And `~/.aws/config`:

```ini
[default]
region = us-east-1
output = json
```

### Method 3: Environment Variables

```bash
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key
export AWS_DEFAULT_REGION=us-east-1
```

### Method 4: IAM Role (EC2/ECS/Lambda)

If running on AWS infrastructure, attach an IAM role with the following policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeRegions",
        "ec2:DescribeInstanceTypes",
        "ec2:DescribeInstanceTypeOfferings"
      ],
      "Resource": "*"
    }
  ]
}
```

## Verification

Test your installation:

```bash
# Check version
truffle --help

# Login to AWS (easiest method!)
aws login

# Test AWS connectivity
truffle search m7i.micro --regions us-east-1

# Run with verbose output
truffle search m7i.large --verbose
```

## Troubleshooting

### "AWS credentials not found"

**Easiest fix:**
```bash
# Just login!
aws login
```

**Or verify AWS configuration:**
```bash
# Verify AWS credentials work
aws sts get-caller-identity

# If that fails, login again
aws login
```

**Alternative methods:**
```bash
# Set environment variables
export AWS_ACCESS_KEY_ID=your_access_key
export AWS_SECRET_ACCESS_KEY=your_secret_key

# Or use a profile
export AWS_PROFILE=my-profile
```

### "Permission denied" errors

Ensure your AWS credentials have the required EC2 permissions listed above.

### "Command not found"

Make sure the `truffle` binary is in your PATH:

```bash
# Add to PATH temporarily
export PATH=$PATH:/path/to/truffle

# Or install globally
sudo cp truffle /usr/local/bin/
```

### Build errors

If you encounter build errors, ensure you have the latest Go version:

```bash
# Update Go modules
go mod tidy
go mod download

# Clean and rebuild
go clean
go build
```

## Next Steps

- Read the [README.md](README.md) for usage examples
- Check out [examples/](examples/) for common use cases
- Join our [community discussions](https://github.com/yourusername/truffle/discussions)

## Uninstallation

```bash
# If installed via make
sudo make clean
sudo rm /usr/local/bin/truffle

# Manual removal
sudo rm /usr/local/bin/truffle
```
