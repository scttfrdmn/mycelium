# DNS Setup Guide for spore.host

This document tracks the DNS configuration for automatic instance hostnames.

## Architecture

This is an **open source project**, so anyone can use spawn and get automatic DNS. The security model uses a centralized Lambda API Gateway that validates instance identity before updating DNS.

**Default Account** (Infrastructure provider - Account: 752123829273):
- Hosts spawnd/spored binaries in S3
- Route53 hosted zone for spore.host
- Lambda DNS updater (Go-based)
- API Gateway public endpoint

**User Accounts** (any AWS account):
- Spawn instances with `spawn:managed` tag
- Call public API to update DNS
- No special IAM roles needed!

## Route53 Hosted Zone

**Domain**: spore.host
**Hosted Zone ID**: Z048907324UNXKEK9KX93
**AWS Account**: 752123829273 (default)
**Created**: 2025-12-22

## AWS Nameservers

Configure these nameservers at your domain registrar:

```
ns-1774.awsdns-29.co.uk
ns-422.awsdns-52.com
ns-816.awsdns-38.net
ns-1091.awsdns-08.org
```

## API Gateway Endpoint

**Public Endpoint**: `https://f4gm19tl70.execute-api.us-east-1.amazonaws.com/prod/update-dns`
**Method**: POST
**Authentication**: None (validated via instance identity)
**Region**: us-east-1

## Nameserver Update Instructions

### If you purchased from Namecheap:
1. Log in to Namecheap
2. Go to Domain List → Manage for spore.host
3. Click "Advanced DNS" tab
4. Under "Nameservers", select "Custom DNS"
5. Enter the 4 AWS nameservers listed above
6. Save changes
7. Wait 24-48 hours for DNS propagation (usually faster)

### If you purchased from another registrar:
1. Log in to your registrar's control panel
2. Find DNS/Nameserver settings for spore.host
3. Change from default nameservers to custom nameservers
4. Enter the 4 AWS nameservers listed above
5. Save changes

## Verify DNS Propagation

After updating nameservers, verify propagation:

```bash
# Check nameservers (should show AWS nameservers)
dig NS spore.host

# Or use online tools:
# https://www.whatsmydns.net/#NS/spore.host
```

## Test DNS Resolution

Once propagated, test with a sample record:

```bash
# Create a test A record
AWS_PROFILE=aws aws route53 change-resource-record-sets \
  --hosted-zone-id Z08811711PGNZG45042A5 \
  --change-batch '{
    "Changes": [{
      "Action": "CREATE",
      "ResourceRecordSet": {
        "Name": "test.spore.host",
        "Type": "A",
        "TTL": 60,
        "ResourceRecords": [{"Value": "1.2.3.4"}]
      }
    }]
  }'

# Wait a minute, then test resolution
dig test.spore.host
```

## Security Model (API Gateway + Lambda)

This system is designed for **open source** use - anyone can call the API from any AWS account.

### How It Works

1. **Spawn instance** launches with `spawn:managed=true` tag
2. **Instance gets identity document** from EC2 IMDS (signed by AWS)
3. **Instance calls API Gateway** with:
   - Instance identity document (base64-encoded)
   - Instance identity signature (base64-encoded)
   - Desired DNS record name
   - IP address
4. **Lambda validates**:
   - ✅ Instance identity signature (cryptographically verified with AWS public key)
   - ✅ Instance exists and has `spawn:managed` tag (via DescribeInstances)
   - ✅ IP address matches instance public IP
   - ✅ Record name format is valid
5. **Lambda updates** Route53 DNS record
6. **Full audit trail** via CloudWatch Logs

### Security Guarantees

✅ **No shared secrets**: Instance identity document can't be forged without AWS private key
✅ **Per-instance validation**: Each request verified against live AWS instance metadata
✅ **IP verification**: DNS only updated if IP matches instance's actual public IP
✅ **Tag enforcement**: Only instances with `spawn:managed` tag can update DNS
✅ **No IAM roles needed**: User accounts don't need any special permissions
✅ **Audit trail**: All requests logged in CloudWatch

### API Request Format

```bash
POST https://f4gm19tl70.execute-api.us-east-1.amazonaws.com/prod/update-dns
Content-Type: application/json

{
  "instance_identity_document": "<base64-encoded-document>",
  "instance_identity_signature": "<base64-encoded-signature>",
  "record_name": "my-instance",
  "ip_address": "1.2.3.4",
  "action": "UPSERT"  // or "DELETE"
}
```

### Getting Instance Identity

From within an EC2 instance:

```bash
# Get instance identity document
IDENTITY_DOC=$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/document | base64 -w0)

# Get instance identity signature
IDENTITY_SIG=$(curl -s http://169.254.169.254/latest/dynamic/instance-identity/signature)

# Get public IP
PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4)

# Update DNS
curl -X POST https://f4gm19tl70.execute-api.us-east-1.amazonaws.com/prod/update-dns \
  -H "Content-Type: application/json" \
  -d "{
    \"instance_identity_document\": \"$IDENTITY_DOC\",
    \"instance_identity_signature\": \"$IDENTITY_SIG\",
    \"record_name\": \"my-instance\",
    \"ip_address\": \"$PUBLIC_IP\",
    \"action\": \"UPSERT\"
  }"
```

### Lambda Function

**Name**: spawn-dns-updater
**Runtime**: Go (provided.al2023)
**Source**: `spawn/lambda/dns-updater/`
**IAM Role**: SpawnDNSLambdaExecutionRole

**Permissions**:
- Route53: Update records in spore.host zone only
- EC2: DescribeInstances (to validate instance metadata)
- CloudWatch Logs: Write logs

## DNSSEC Configuration

**Status**: ✅ Enabled and Signing

DNSSEC adds cryptographic signatures to DNS records to prevent DNS hijacking and cache poisoning attacks.

### Configuration Details

**KMS Key**: `arn:aws:kms:us-east-1:752123829273:key/b638147e-f2c0-48bd-a3a6-5f1b7d4773d0`
- Key Type: ECC_NIST_P256
- Purpose: SIGN_VERIFY
- Algorithm: ECDSAP256SHA256

**Key Signing Key (KSK)**: spore-host-ksk
- Key Tag: 12735
- Algorithm: 13 (ECDSAP256SHA256)
- Digest Type: 2 (SHA-256)

**DS Record** (added to Porkbun):
```
12735 13 2 0179EFB5FA92E41D46256E7C1D8628B9DD7C0529E85E400F9B48213685BBA5E4
```

### Security Benefits

🔒 **Prevents DNS hijacking** - Cryptographically proves DNS records are authentic
🔒 **Blocks cache poisoning** - Malicious DNS cache entries are rejected
🔒 **Protects SSH connections** - Users connecting to `*.spore.host` get authentic IPs
🔒 **Chain of trust** - DNSSEC signatures verified from root DNS down to spore.host

### Verification

Check DNSSEC status:

```bash
# Check for DNSSEC signatures
dig +dnssec spore.host SOA

# Validate DNSSEC chain
delv spore.host

# Online validators
https://dnssec-debugger.verisignlabs.com/spore.host
https://dnsviz.net/d/spore.host/dnssec/
```

### Key Rotation

Route53 automatically rotates Zone Signing Keys (ZSK). The Key Signing Key (KSK) is stable and rarely needs rotation.

If KSK rotation is needed:
1. Create new KSK in Route53
2. Get new DS record
3. Add new DS record to Porkbun
4. Wait 48 hours for propagation
5. Remove old DS record from Porkbun
6. Deactivate old KSK in Route53

## Next Steps

- [x] Create Route53 hosted zone
- [x] Create Lambda DNS updater (Go)
- [x] Create API Gateway endpoint
- [x] Update nameservers at registrar
- [x] Enable DNSSEC
- [x] Add DS record to registrar
- [ ] Verify DNS propagation (24-48 hours)
- [ ] Implement `spawn launch --dns` flag
- [ ] Implement spawnd/spored DNS update logic (call API on launch/IP change)
- [ ] Add `spawn dns` management commands (list, update, delete)

## Useful Commands

```bash
# List all records in the zone
AWS_PROFILE=aws aws route53 list-resource-record-sets \
  --hosted-zone-id Z08811711PGNZG45042A5

# Get hosted zone details
AWS_PROFILE=aws aws route53 get-hosted-zone \
  --id Z08811711PGNZG45042A5
```
