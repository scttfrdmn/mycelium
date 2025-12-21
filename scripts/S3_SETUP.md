# S3 Bucket Setup for spawnd Distribution

This guide explains how to set up the central S3 buckets for distributing spawnd binaries.

## Architecture

```
┌─────────────────────────────────────┐
│  Central "Spawn" AWS Account        │
│                                     │
│  S3 Buckets (Public Read):          │
│  - spawn-binaries-us-east-1         │
│  - spawn-binaries-us-west-2         │
│  - spawn-binaries-eu-west-1         │
│  - ... (one per region)             │
│                                     │
│  Contents:                          │
│  - spawnd-linux-amd64               │
│  - spawnd-linux-amd64.sha256        │
│  - spawnd-linux-arm64               │
│  - spawnd-linux-arm64.sha256        │
└─────────────────────────────────────┘
         ↑
         │ Public HTTPS download
         │ SHA256 checksum verification
         │
    ┌────┴─────┐
    │ EC2      │
    │ Instance │
    │ (any     │
    │ account) │
    └──────────┘
```

## Security Model

- **Public S3 Buckets**: Anyone can download spawnd binaries
- **SHA256 Verification**: Integrity verified cryptographically on instance
- **No Authentication**: Standard practice for open-source binaries
- **Versioning Enabled**: Old versions kept for 90 days
- **Regional Buckets**: Low-latency downloads with us-east-1 fallback

## Prerequisites

1. AWS CLI installed and configured
2. AWS account for hosting binaries (different from user accounts)
3. AWS profile configured for the central account

## Setup Steps

### 1. Configure AWS Profile

```bash
# Add profile for your central spawn account
aws configure --profile my-spawn-account

# Verify
aws sts get-caller-identity --profile my-spawn-account
```

### 2. Build spawnd Binaries

```bash
cd /Users/scttfrdmn/src/mycelium
make build-all

# Verify binaries exist
ls -lh spawn/bin/spawnd-*
```

### 3. Create S3 Buckets

Create buckets in all regions:

```bash
./scripts/setup_s3_buckets.sh my-spawn-account all
```

Or specific regions:

```bash
./scripts/setup_s3_buckets.sh my-spawn-account us-east-1 us-west-2 eu-west-1
```

This script:
- Creates `spawn-binaries-{region}` buckets
- Configures public read access for `spawnd-*` objects
- Enables versioning
- Sets up lifecycle policies (90-day retention for old versions)

### 4. Upload Binaries

```bash
./scripts/upload_spawnd.sh my-spawn-account all
```

Or specific regions:

```bash
./scripts/upload_spawnd.sh my-spawn-account us-east-1 us-west-2
```

This script:
- Generates SHA256 checksums for each binary
- Uploads binaries and checksums to all regional buckets
- Sets appropriate metadata

### 5. Verify

Test download and verification:

```bash
# Download binary
curl -O https://spawn-binaries-us-east-1.s3.amazonaws.com/spawnd-linux-amd64

# Download checksum
curl -O https://spawn-binaries-us-east-1.s3.amazonaws.com/spawnd-linux-amd64.sha256

# Verify
echo "$(cat spawnd-linux-amd64.sha256)  spawnd-linux-amd64" | sha256sum --check
```

Expected output: `spawnd-linux-amd64: OK`

## Updating Binaries

When releasing a new version:

```bash
# 1. Update version in code
# 2. Rebuild binaries
make build-all

# 3. Upload to all regions
./scripts/upload_spawnd.sh my-spawn-account all
```

Versioning is enabled, so old versions are automatically kept for 90 days.

## How It Works

When spawn launches an EC2 instance, the user-data script:

1. **Detects architecture** (AMD64 or ARM64)
2. **Detects region** from EC2 metadata
3. **Downloads binary** from regional bucket (or us-east-1 fallback)
4. **Downloads SHA256 checksum**
5. **Verifies checksum** - fails if mismatch
6. **Installs spawnd** if verification passes

See `spawn/cmd/launch.go` lines 388-460 for implementation.

## Bucket Structure

```
spawn-binaries-us-east-1/
├── spawnd-linux-amd64          # 15MB binary
├── spawnd-linux-amd64.sha256   # 64-char hex checksum
├── spawnd-linux-arm64          # 14MB binary
└── spawnd-linux-arm64.sha256   # 64-char hex checksum
```

## Costs

Estimated monthly costs (per region):

- **Storage**: $0.023/GB → ~$0.70/month for 30GB (versioning)
- **Data Transfer**:
  - First 10TB: $0.09/GB
  - 1000 downloads/month (15MB each) = 15GB = ~$1.35
- **Requests**: $0.0004/1000 GET requests → negligible
- **Total per region**: ~$2/month
- **10 regions**: ~$20/month

## Security Considerations

### Why Public Buckets?

- **Industry Standard**: How apt, yum, GitHub, Docker Hub distribute
- **Simplicity**: Works across any AWS account, on-prem, other clouds
- **Integrity**: SHA256 prevents tampering
- **No Credentials**: No secrets in user-data or code

### What's Protected?

- ✅ **Integrity**: SHA256 verification prevents modified binaries
- ✅ **Authenticity**: Only you can upload to your buckets
- ✅ **Availability**: Regional buckets + versioning

### What's Not Protected?

- ⚠️ **Confidentiality**: Anyone can download binaries
  - This is intentional - spawnd is an open-source utility
  - No proprietary code or secrets in the binary

## Troubleshooting

### Bucket Already Exists
If you see "bucket already exists", that's fine - the script continues.

### Permission Denied
Verify your AWS profile has these permissions:
- `s3:CreateBucket`
- `s3:PutBucketPolicy`
- `s3:PutBucketVersioning`
- `s3:PutObject`

### Checksum Verification Fails on Instance
- Check that SHA256 file was uploaded
- Verify file wasn't corrupted during upload
- Re-upload: `./scripts/upload_spawnd.sh my-spawn-account us-east-1`

### Regional Bucket Not Found
Instances will automatically fallback to us-east-1. For best performance, ensure buckets exist in all regions where you launch instances.

## Cleanup

To remove all buckets (WARNING: destroys all binaries):

```bash
# List regions
REGIONS=(us-east-1 us-west-2 eu-west-1)

# Delete buckets
for region in "${REGIONS[@]}"; do
    bucket="spawn-binaries-${region}"
    echo "Deleting $bucket..."
    aws s3 rb "s3://${bucket}" --force --profile my-spawn-account --region "$region"
done
```
