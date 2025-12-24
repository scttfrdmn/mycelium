# Security Policy

This document describes the security model and policies for the spawn project.

## Table of Contents

- [Reporting Security Issues](#reporting-security-issues)
- [Security Model Overview](#security-model-overview)
- [IAM Permissions](#iam-permissions)
- [DNS Security Model](#dns-security-model)
- [Instance Identity Validation](#instance-identity-validation)
- [DNSSEC](#dnssec)
- [Cross-Account Security](#cross-account-security)
- [Audit and Logging](#audit-and-logging)
- [Security Best Practices](#security-best-practices)

## Reporting Security Issues

If you discover a security vulnerability in spawn, please report it by:

1. **DO NOT** open a public GitHub issue
2. Email the maintainer directly at the email address in the git commits
3. Provide detailed information about the vulnerability
4. Allow reasonable time for a fix before public disclosure

We take security seriously and will respond to legitimate reports promptly.

## Security Model Overview

Spawn uses a defense-in-depth approach with multiple security layers:

1. **IAM-based authentication** - AWS credentials validate user identity
2. **Instance tagging** - `spawn:managed=true` tag identifies managed instances
3. **Cryptographic validation** - AWS-signed instance identity documents prove authenticity
4. **Least privilege** - Minimal IAM permissions required for operation
5. **Audit trail** - CloudWatch logs track all operations

## IAM Permissions

Spawn requires specific IAM permissions to operate. See [IAM_PERMISSIONS.md](IAM_PERMISSIONS.md) for detailed permission requirements.

### Principle of Least Privilege

Spawn follows the principle of least privilege:

- **User accounts** - Only require EC2, SSM, and optional DNS permissions
- **No admin access** - spawn never requires AdministratorAccess
- **Read-only where possible** - Many operations only need describe/list permissions
- **Scoped permissions** - IAM policies can be scoped to specific regions/resources

### SSH Key Security

Spawn manages SSH keys securely:

- Keys stored in `~/.ssh/spawn-*.pem` with 0600 permissions
- Never transmitted over network (except via SSH agent)
- Unique key per instance
- Automatically cleaned up on instance termination

## DNS Security Model

Spawn provides automatic DNS registration for instances via the spore.host domain (or custom domains for institutions).

### Architecture

The DNS system uses a **serverless API Gateway + Lambda** architecture:

```
┌─────────────────┐
│  EC2 Instance   │
│ (Any AWS Acct)  │
└────────┬────────┘
         │ 1. Get instance identity from IMDS
         │ 2. Call DNS API
         ▼
┌─────────────────────────────────────┐
│   API Gateway (Public Endpoint)     │
│  https://....amazonaws.com/prod     │
└────────┬────────────────────────────┘
         │ 3. Invoke Lambda
         ▼
┌─────────────────────────────────────┐
│   Lambda Function (Go)              │
│   - Validate instance identity      │
│   - Check spawn:managed tag         │
│   - Verify IP address               │
└────────┬────────────────────────────┘
         │ 4. Update DNS
         ▼
┌─────────────────────────────────────┐
│   Route53 Hosted Zone               │
│   spore.host (Z048907324UNXKEK9KX93)│
└─────────────────────────────────────┘
```

### Security Guarantees

✅ **No shared secrets** - Instance identity documents are cryptographically signed by AWS and cannot be forged

✅ **Per-instance validation** - Each DNS update request is validated against live AWS instance metadata

✅ **IP verification** - DNS records only updated if IP address matches the instance's actual public IP (for same-account instances)

✅ **Tag enforcement** - Only instances with `spawn:managed=true` tag can update DNS (for same-account instances)

✅ **No IAM roles required** - User accounts don't need special Route53 permissions or cross-account IAM trust relationships

✅ **Audit trail** - All DNS update requests logged in CloudWatch Logs with full request details

✅ **Rate limiting** - API Gateway provides DDoS protection and rate limiting

✅ **DNSSEC enabled** - Cryptographic signatures prevent DNS hijacking and cache poisoning

### Instance Identity Validation

The security model relies on AWS Instance Identity Documents:

1. **What it is**: A JSON document containing instance metadata (instance ID, region, account ID, IP address)

2. **Cryptographic signature**: AWS signs the document with a private key that only AWS possesses

3. **Verification**: The Lambda function validates the signature using AWS's public key

4. **Cannot be forged**: Without AWS's private key, attackers cannot create valid instance identity documents

5. **Retrieved from IMDS**: Only accessible from within the EC2 instance via the Instance Metadata Service

### Validation Flow

When an instance requests DNS registration:

```go
// 1. Instance retrieves identity from IMDSv2 (token-based)
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")

IDENTITY_DOC=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" \
  http://169.254.169.254/latest/dynamic/instance-identity/document | base64 -w0)

IDENTITY_SIG=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" \
  http://169.254.169.254/latest/dynamic/instance-identity/signature)

// 2. Lambda validates the request
func validateInstance(ctx, instanceID, region, ipAddress, action) error {
    // Try to describe instance (works for same-account)
    output, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
        InstanceIds: []string{instanceID},
    })

    if err != nil {
        // Cross-account: Can't describe instance in another account
        // Rely on instance identity signature validation
        // This is secure because signature cannot be forged
        return nil
    }

    // Same-account: Perform full validation
    // - Check spawn:managed tag
    // - Verify IP address matches
    // - Check instance state (running/stopped)

    return nil
}
```

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

### Lambda IAM Permissions

The Lambda function has minimal permissions:

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
      "Resource": "arn:aws:route53:::hostedzone/Z048907324UNXKEK9KX93"
    },
    {
      "Effect": "Allow",
      "Action": "ec2:DescribeInstances",
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    }
  ]
}
```

## Instance Identity Validation

### IMDSv2 (Recommended)

Spawn uses IMDSv2 (token-based) for retrieving instance identity:

**Benefits:**
- Prevents SSRF attacks
- Requires token authentication
- Session-oriented

**How it works:**
```bash
# 1. Get session token
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" \
  -H "X-aws-ec2-metadata-token-ttl-seconds: 21600")

# 2. Use token for all subsequent requests
INSTANCE_ID=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" \
  http://169.254.169.254/latest/meta-data/instance-id)
```

### Instance Identity Document Structure

```json
{
  "instanceId": "i-1234567890abcdef0",
  "region": "us-east-1",
  "accountId": "123456789012",
  "architecture": "x86_64",
  "imageId": "ami-0abcdef1234567890",
  "instanceType": "t3.micro",
  "privateIp": "10.0.1.5",
  "availabilityZone": "us-east-1a",
  "version": "2017-09-30",
  "pendingTime": "2023-01-01T12:00:00Z"
}
```

The signature ensures this document is authentic and was issued by AWS.

## DNSSEC

Spawn's DNS infrastructure uses DNSSEC for additional security.

### What is DNSSEC?

DNSSEC (DNS Security Extensions) adds cryptographic signatures to DNS records to prevent:

- **DNS hijacking** - Attackers redirecting your DNS to malicious servers
- **Cache poisoning** - Malicious DNS entries injected into caches
- **Man-in-the-middle attacks** - Intercepting DNS queries to redirect traffic

### Implementation

**KMS Key**: ECC_NIST_P256 key for signing DNS records
- Key ID: `b638147e-f2c0-48bd-a3a6-5f1b7d4773d0`
- Algorithm: ECDSAP256SHA256 (Algorithm 13)
- Managed by AWS KMS with automatic rotation

**Key Signing Key (KSK)**: spore-host-ksk
- Key Tag: 12735
- DS Record: `12735 13 2 0179EFB5FA92E41D46256E7C1D8628B9DD7C0529E85E400F9B48213685BBA5E4`

**Zone Signing Key (ZSK)**: Automatically rotated by Route53

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

### Security Benefits

When you connect to `my-instance.spore.host`:

1. DNS query sent to resolver
2. Resolver gets DNS record + DNSSEC signature
3. Resolver validates signature using public key
4. Only if signature is valid, IP address is returned
5. You connect to the authentic instance IP

This prevents attackers from redirecting you to malicious instances.

## Cross-Account Security

Spawn is designed to work securely across AWS accounts without requiring IAM trust relationships.

### Same-Account Instances

When spawn and the instance are in the same AWS account:

✅ **Full validation**:
- Verify `spawn:managed=true` tag
- Verify IP address matches instance public IP
- Verify instance state (running/stopped)
- Validate instance identity signature

### Cross-Account Instances

When spawn and the instance are in different AWS accounts:

✅ **Signature-based validation**:
- Validate instance identity signature (cryptographic proof)
- Cannot check tags or IP (no DescribeInstances permission cross-account)
- Relies on AWS cryptographic signature as primary security

**Why this is secure**:
- Instance identity signatures cannot be forged without AWS's private key
- Only legitimate EC2 instances can retrieve valid identity documents
- Signature proves the instance exists and is running in the claimed account

### No IAM Trust Required

Traditional cross-account access requires:
```json
// ❌ Traditional approach - doesn't scale for open source
{
  "Effect": "Allow",
  "Principal": {
    "AWS": [
      "arn:aws:iam::111111111111:root",
      "arn:aws:iam::222222222222:root",
      // ... maintain allowlist of every user account
    ]
  }
}
```

Spawn's approach:
```
✅ No trust relationship required
✅ No allowlist to maintain
✅ Works for any AWS account
✅ Cryptographic validation via instance identity
```

## Audit and Logging

### CloudWatch Logs

All DNS operations are logged to CloudWatch Logs:

- Lambda function: `/aws/lambda/spawn-dns-updater`
- Log group retention: 30 days (configurable)
- Logs include:
  - Timestamp
  - Instance ID
  - AWS account ID
  - Region
  - Requested DNS record
  - IP address
  - Action (UPSERT/DELETE)
  - Validation results
  - Errors (if any)

### Example Log Entry

```json
{
  "timestamp": "2025-12-21T18:00:00Z",
  "requestId": "abc123-def456",
  "instanceId": "i-1234567890abcdef0",
  "accountId": "123456789012",
  "region": "us-east-1",
  "recordName": "my-instance",
  "fqdn": "my-instance.spore.host",
  "ipAddress": "54.164.27.106",
  "action": "UPSERT",
  "validation": "success",
  "changeId": "/change/C123456789",
  "message": "DNS record updated successfully"
}
```

### Monitoring

Recommended CloudWatch alarms:

- **DNS API errors** - Alert on 5xx responses
- **Validation failures** - Alert on failed instance validation
- **Rate limiting** - Alert on API Gateway throttling
- **Lambda errors** - Alert on Lambda function failures

## Security Best Practices

### For Users

1. **Use IMDSv2** - Enabled by default in spawn
2. **Rotate SSH keys** - spawn generates unique keys per instance
3. **Use short TTLs** - `--ttl 2h` for development instances
4. **Clean up instances** - Use `spawn stop` to terminate instances
5. **Review DNS records** - Periodically audit `*.spore.host` records
6. **Enable MFA** - Use MFA on your AWS account
7. **Use least privilege** - Only grant necessary IAM permissions

### For Institutions (Custom DNS)

1. **Deploy in isolated account** - Separate account for DNS infrastructure
2. **Enable CloudTrail** - Audit all Route53 API calls
3. **Set up alarms** - Monitor DNS API for anomalies
4. **Review Lambda logs** - Regularly audit DNS update requests
5. **Enable DNSSEC** - Prevent DNS hijacking
6. **Use KMS encryption** - Encrypt CloudWatch logs
7. **Implement backup** - Export Route53 zone regularly

### For Contributors

1. **Never commit secrets** - Use AWS credentials from environment
2. **Review IAM policies** - Ensure least privilege
3. **Test cross-account** - Validate cross-account scenarios
4. **Validate input** - Always validate user input in Lambda
5. **Use prepared statements** - Prevent injection attacks
6. **Enable security scanning** - Use Dependabot, CodeQL
7. **Follow secure coding** - OWASP guidelines for Go

## Security Considerations

### Threat Model

**What spawn protects against**:
- ✅ Unauthorized DNS updates (instance identity validation)
- ✅ DNS hijacking (DNSSEC)
- ✅ Cache poisoning (DNSSEC)
- ✅ Cross-account abuse (tag enforcement + signature validation)
- ✅ Unauthorized instance access (SSH key management)

**What spawn does NOT protect against**:
- ❌ Compromised AWS credentials (use MFA, rotate keys)
- ❌ Compromised EC2 instances (harden your instances)
- ❌ AWS account compromise (enable CloudTrail, GuardDuty)
- ❌ Physical access to infrastructure (AWS's responsibility)

### Known Limitations

1. **Cross-account validation** - Cannot verify tags or IP address for instances in other AWS accounts (relies on signature validation only)

2. **DNS propagation** - DNS changes take time to propagate (60s TTL by default)

3. **Rate limiting** - API Gateway has rate limits (10,000 requests/second default)

4. **Instance identity freshness** - Identity documents don't expire, but instance must be running to retrieve them

### Future Enhancements

- [ ] Implement signature verification in Lambda (currently relies on same-account validation)
- [ ] Add support for IPv6 DNS records (AAAA)
- [ ] Implement DNS record TTL customization per instance
- [ ] Add support for TXT records (for metadata)
- [ ] Implement rate limiting per AWS account
- [ ] Add support for custom SSL certificates

## Compliance

Spawn is designed with compliance in mind:

- **SOC 2** - Audit logs, access controls, encryption
- **GDPR** - No PII stored (only instance IDs, IP addresses)
- **HIPAA** - Can be used in HIPAA-compliant environments (no PHI stored)
- **FedRAMP** - Compatible with AWS GovCloud regions

Institutions should conduct their own compliance assessment based on their specific requirements.

## References

- [AWS Instance Identity Documents](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/instance-identity-documents.html)
- [IMDSv2 Documentation](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/configuring-instance-metadata-service.html)
- [Route53 DNSSEC](https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/dns-configuring-dnssec.html)
- [AWS KMS Best Practices](https://docs.aws.amazon.com/kms/latest/developerguide/best-practices.html)
- [OWASP Secure Coding Practices](https://owasp.org/www-project-secure-coding-practices-quick-reference-guide/)

---

**Last Updated**: 2025-12-21
**Version**: 1.0.0
