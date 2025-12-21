# S3 Setup - Quick Start

## TL;DR

```bash
# 1. Configure AWS profile for your central account
aws configure --profile my-spawn-account

# 2. Build binaries
make build-all

# 3. Create S3 buckets in all regions
./scripts/setup_s3_buckets.sh my-spawn-account all

# 4. Upload binaries with checksums
./scripts/upload_spawnd.sh my-spawn-account all

# 5. Test
curl -O https://spawn-binaries-us-east-1.s3.amazonaws.com/spawnd-linux-amd64
curl -O https://spawn-binaries-us-east-1.s3.amazonaws.com/spawnd-linux-amd64.sha256
echo "$(cat spawnd-linux-amd64.sha256)  spawnd-linux-amd64" | sha256sum --check
```

## What Gets Created

### S3 Buckets (one per region)
- `spawn-binaries-us-east-1`
- `spawn-binaries-us-east-2`
- `spawn-binaries-us-west-1`
- `spawn-binaries-us-west-2`
- `spawn-binaries-eu-west-1`
- `spawn-binaries-eu-west-2`
- `spawn-binaries-eu-central-1`
- `spawn-binaries-ap-southeast-1`
- `spawn-binaries-ap-southeast-2`
- `spawn-binaries-ap-northeast-1`

### Contents per Bucket
```
spawnd-linux-amd64          # Binary (~15MB)
spawnd-linux-amd64.sha256   # Checksum (64 hex chars)
spawnd-linux-arm64          # Binary (~14MB)
spawnd-linux-arm64.sha256   # Checksum (64 hex chars)
```

### Bucket Configuration
- ✅ Public read access for `spawnd-*` objects
- ✅ Versioning enabled
- ✅ Lifecycle: Keep old versions for 90 days
- ✅ Regional endpoints for low-latency downloads

## How Instances Use It

When you run `spawn launch`, the EC2 instance:

1. Detects its architecture (AMD64 or ARM64)
2. Detects its region from metadata
3. Downloads `spawnd-linux-{arch}` from regional bucket
4. Downloads `spawnd-linux-{arch}.sha256` checksum
5. Verifies: `sha256sum --check`
6. ❌ **Fails if checksum doesn't match**
7. ✅ Installs if verification passes

## Security

**Public Distribution Model:**
- Anyone can download binaries (like apt, yum, GitHub)
- SHA256 verification prevents tampering
- No credentials in user-data
- Industry standard for open-source tools

**What's Protected:**
- ✅ Integrity (SHA256 verification)
- ✅ Authenticity (only you can upload to your buckets)
- ✅ Availability (regional buckets + versioning)

**What's Not Protected:**
- ⚠️ Confidentiality (public download)
  - Intentional design for open-source utility

## Updating Binaries

When you release a new version:

```bash
# 1. Update code and rebuild
make build-all

# 2. Upload to all regions (overwrites current, keeps old versions)
./scripts/upload_spawnd.sh my-spawn-account all

# Done! All new instances get the new version
```

Old versions automatically deleted after 90 days (lifecycle policy).

## Costs

~$20/month for 10 regions with moderate usage:
- Storage: ~$0.70/region/month
- Transfer: ~$1.35/region/month for 1000 downloads
- Requests: negligible

## Troubleshooting

### "Bucket already exists"
Normal - script continues and configures it.

### "Access Denied"
Check your AWS profile has S3 permissions.

### "Checksum verification failed" on instance
Re-upload binaries:
```bash
./scripts/upload_spawnd.sh my-spawn-account us-east-1
```

### Want only specific regions?
```bash
# Setup
./scripts/setup_s3_buckets.sh my-spawn-account us-east-1 us-west-2

# Upload
./scripts/upload_spawnd.sh my-spawn-account us-east-1 us-west-2
```

## Full Documentation

See `scripts/S3_SETUP.md` for detailed architecture, security model, and advanced configuration.
