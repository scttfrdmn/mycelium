# Claude Development Notes

## AWS Account Configuration

**CRITICAL**: Resources are split across two AWS accounts:

- **`default` profile** (Account: 752123829273)
  - S3 buckets (spawn-binaries-*)
  - Lambda functions (spawn-dns-updater)
  - Route53 DNS (spore.host hosted zone)
  - Infrastructure resources
  - **NO EC2 instances**

- **`aws` profile** (Account: 942542972736)
  - **ALL EC2 instance provisioning**
  - Test instances
  - Development/testing instances
  - **NO infrastructure resources**

**Cross-Account Requirements**:
- EC2 instances in 'aws' account need access to:
  - S3 bucket in 'default' account (for spored binary downloads)
  - Lambda DNS API in 'default' account (for DNS registration)

**Usage Examples**:
```bash
# Upload spored binary to S3 (use default profile)
aws s3 cp bin/spored s3://spawn-binaries-us-east-1/spored-linux-amd64

# Launch instances (use aws profile ONLY)
AWS_PROFILE=aws ./bin/spawn launch --instance-type t3.micro ...

# Deploy Lambda function (use default profile)
aws lambda update-function-code --function-name spawn-dns-updater ...
```

## DNS Implementation

DNS uses base36-encoded account IDs for subdomain isolation:
- Format: `<name>.<account-base36>.spore.host`
- Account 752123829273 → Base36: `c0zxr0ao`
- Example: `test-base36.c0zxr0ao.spore.host`

## Spored Agent

The spored agent runs on EC2 instances and handles:
- Automatic DNS registration on startup
- DNS cleanup on termination (SIGTERM, TTL, idle, Spot interruption)
- Idle detection and auto-termination
- Hibernation support
