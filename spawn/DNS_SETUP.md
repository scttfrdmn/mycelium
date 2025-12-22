# DNS Setup Guide for spore.host

This document tracks the DNS configuration for automatic instance hostnames.

## Route53 Hosted Zone

**Domain**: spore.host
**Hosted Zone ID**: Z08811711PGNZG45042A5
**AWS Account**: (configured with AWS_PROFILE=aws)
**Created**: 2025-12-22

## AWS Nameservers

Configure these nameservers at your domain registrar:

```
ns-1770.awsdns-29.co.uk
ns-1294.awsdns-33.org
ns-771.awsdns-32.net
ns-260.awsdns-32.com
```

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

## Cross-Account IAM Role

**Role Name**: SpawnDNSUpdaterRole
**ARN**: `arn:aws:iam::942542972736:role/SpawnDNSUpdaterRole`
**AWS Account**: 942542972736

### Security Model

✅ **Trust Policy**: Only allows principals with `spawn:managed` tag to assume role
✅ **Permissions**: Limited to Route53 operations on spore.host zone (Z08811711PGNZG45042A5)
✅ **Condition**: Records must match pattern `*.spore.host`
✅ **Temporary Credentials**: No long-lived secrets, credentials via AssumeRole

### How It Works

1. Spawn instances are launched with `spawn:managed=true` tag
2. Instance IAM role has permission to assume SpawnDNSUpdaterRole
3. spawnd assumes the role to get temporary credentials
4. spawnd updates DNS record (e.g., `my-instance.spore.host`)
5. Can only update records matching the instance name
6. Full CloudTrail audit trail of all DNS changes

### Trust Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {"AWS": "*"},
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "aws:PrincipalTag/spawn:managed": "true"
        }
      }
    }
  ]
}
```

### Permissions Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "route53:ChangeResourceRecordSets",
        "route53:ListResourceRecordSets"
      ],
      "Resource": "arn:aws:route53:::hostedzone/Z08811711PGNZG45042A5",
      "Condition": {
        "StringLike": {
          "route53:ChangeResourceRecordSetsNormalizedRecordNames": ["*.spore.host"]
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": ["route53:GetChange"],
      "Resource": "arn:aws:route53:::change/*"
    }
  ]
}
```

See ROADMAP.md Phase 2 item #7 for full security model.

## Next Steps

- [ ] Update nameservers at registrar
- [ ] Verify DNS propagation (24-48 hours)
- [ ] Create cross-account IAM role for DNS updates
- [ ] Implement `spawn launch --dns` flag
- [ ] Implement spawnd DNS update logic
- [ ] Add `spawn dns` management commands

## Useful Commands

```bash
# List all records in the zone
AWS_PROFILE=aws aws route53 list-resource-record-sets \
  --hosted-zone-id Z08811711PGNZG45042A5

# Get hosted zone details
AWS_PROFILE=aws aws route53 get-hosted-zone \
  --id Z08811711PGNZG45042A5
```
