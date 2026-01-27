# NIST 800-171 Rev 3 Compliance Quickstart

## Overview

spawn v0.14.0 introduces compliance mode support for NIST 800-171 Rev 3, enabling government and regulated organizations to use spawn while meeting Controlled Unclassified Information (CUI) protection requirements.

## What is NIST 800-171?

NIST SP 800-171 Revision 3 provides security requirements for protecting Controlled Unclassified Information (CUI) in nonfederal systems and organizations. It contains 110 security controls across 14 families.

## Quick Start

### Enable NIST 800-171 Compliance

Launch an instance with compliance mode enabled:

```bash
spawn launch \
  --instance-type t3.micro \
  --ttl 2h \
  --nist-800-171
```

### What Gets Enforced

When you enable `--nist-800-171`, spawn automatically enforces:

- **EBS Encryption (SC-28)**: All EBS volumes encrypted at rest using AWS KMS
- **IMDSv2 Required (AC-17)**: Instance metadata service requires authentication tokens
- **Audit Logging (AU-02)**: Structured audit logs generated for all operations
- **Security Groups (SC-07)**: Boundary protection via AWS security groups
- **TLS Encryption (SC-08, SC-13)**: All API communications use TLS 1.2+
- **IAM Authentication (IA-02, IA-05)**: AWS IAM for identity and access management
- **Least Privilege (AC-06)**: IAM roles scoped to minimum required permissions

### Validation

Validate existing instances against NIST 800-171 controls:

```bash
# Validate all spawn-managed instances
spawn validate --nist-800-171

# Validate specific instance
spawn validate --instance-id i-0abc123 --nist-800-171

# Output as JSON for automation
spawn validate --nist-800-171 --output json
```

### Configuration File

Create `~/.spawn/config.yaml` to set compliance mode permanently:

```yaml
compliance:
  mode: "nist-800-171"
  enforce_encrypted_ebs: true
  enforce_imdsv2: true
  audit_logging_required: true
  allow_shared_infrastructure: true  # Show warnings, but allow shared infra
  strict_mode: false  # Show warnings instead of errors

infrastructure:
  mode: "shared"  # Use mycelium-infra account resources (default)
```

### Environment Variables

Set compliance mode via environment variables:

```bash
export SPAWN_COMPLIANCE_MODE=nist-800-171
export SPAWN_COMPLIANCE_ENFORCE_ENCRYPTED_EBS=true
export SPAWN_COMPLIANCE_ENFORCE_IMDSV2=true
export SPAWN_COMPLIANCE_STRICT_MODE=false

spawn launch --instance-type t3.micro --ttl 2h
```

### Strict Mode

Enable strict mode to fail launches on any compliance violations:

```bash
spawn launch \
  --instance-type t3.micro \
  --ttl 2h \
  --nist-800-171 \
  --compliance-strict
```

In strict mode, any configuration that violates compliance controls will cause the launch to fail with clear error messages referencing the specific control IDs.

## Control Mapping

spawn implements the following NIST 800-171 controls:

| Control ID | Name | Implementation |
|------------|------|----------------|
| **AC-06** | Least Privilege | IAM role scoping (v0.13.0) |
| **AC-17** | Remote Access | IMDSv2 enforcement, SSH key management |
| **AU-02** | Event Logging | Structured audit logging (v0.13.0) |
| **IA-02** | Identification and Authentication | AWS IAM authentication |
| **IA-05** | Authenticator Management | KMS secrets encryption (v0.13.0), SSH key pairs |
| **SC-07** | Boundary Protection | Security group configuration |
| **SC-08** | Transmission Confidentiality | TLS for all API calls (AWS SDK) |
| **SC-12** | Cryptographic Key Management | KMS integration (v0.13.0) |
| **SC-13** | Cryptographic Protection | EBS encryption, TLS |
| **SC-28** | Protection at Rest | EBS encryption, S3 encryption |

**Note**: spawn implements **technical controls**. Organizational controls (policies, procedures, training, physical security) remain the customer's responsibility.

## Infrastructure Modes

### Shared Infrastructure (Default)

By default, spawn uses infrastructure resources in the mycelium-infra account (DynamoDB, S3, Lambda). This is suitable for NIST 800-171 compliance, but generates warnings:

```
⚠️  Using shared infrastructure with compliance mode enabled.
    For full compliance, consider deploying self-hosted infrastructure.
    Run 'spawn config init --self-hosted' to configure.
```

### Self-Hosted Infrastructure (Recommended)

For full compliance and data isolation, deploy spawn infrastructure in your own AWS account:

```bash
# Interactive wizard to configure self-hosted infrastructure
spawn config init --self-hosted

# Deploy CloudFormation stack
cd deployment/cloudformation
aws cloudformation create-stack \
  --stack-name spawn-infrastructure \
  --template-body file://self-hosted-stack.yaml \
  --capabilities CAPABILITY_IAM

# Update config with stack outputs
spawn config init --self-hosted
```

See [Self-Hosted Infrastructure Guide](../how-to/self-hosted-infrastructure.md) for detailed instructions.

## Validation Reports

### Text Output (Default)

```bash
$ spawn validate --nist-800-171

Compliance Validation Report (NIST 800-171 Rev 3)
==================================================

Instances Scanned: 12
Compliant: 10
Non-Compliant: 2

Non-Compliant Instances:
  i-0abc123 (my-instance):
    ✗ [SC-28] Protection of Information at Rest: EBS volumes not encrypted
    ✗ [AC-17] Remote Access: IMDSv2 not enforced

  i-0def456 (worker-2):
    ✗ [AC-17] Remote Access: IMDSv2 not enforced

Recommendations:
  1. Terminate and relaunch non-compliant instances with --nist-800-171
  2. Enable default EBS encryption: aws ec2 enable-ebs-encryption-by-default
  3. Review networking configuration for compliance requirements
```

### JSON Output

```bash
$ spawn validate --nist-800-171 --output json

{
  "compliance_mode": "NIST 800-171 Rev 3",
  "instances_scanned": 12,
  "compliant_count": 10,
  "non_compliant_count": 2,
  "total_violations": 3,
  "instances": [
    {
      "instance_id": "i-0abc123",
      "name": "my-instance",
      "region": "us-east-1",
      "type": "t3.micro",
      "state": "running",
      "compliant": false,
      "violations": [
        {
          "control_id": "SC-28",
          "control_name": "Protection of Information at Rest",
          "description": "EBS volumes not encrypted",
          "severity": "high",
          "remediation": ""
        }
      ]
    }
  ]
}
```

## Common Scenarios

### Development/Testing

For development and testing, shared infrastructure with compliance mode is sufficient:

```bash
spawn launch --instance-type t3.micro --ttl 1h --nist-800-171
```

### Production/Regulated Workloads

For production or regulated workloads, use self-hosted infrastructure:

```yaml
# ~/.spawn/config.yaml
compliance:
  mode: "nist-800-171"
  strict_mode: true
  allow_shared_infrastructure: false

infrastructure:
  mode: "self-hosted"
  dynamodb:
    schedules_table: "my-spawn-schedules"
  s3:
    binaries_bucket_prefix: "my-spawn-binaries"
  lambda:
    scheduler_handler_arn: "arn:aws:lambda:us-east-1:123456789012:function:my-spawn-scheduler"
```

### CI/CD Pipelines

Export compliance settings as environment variables in CI/CD:

```yaml
# .gitlab-ci.yml
variables:
  SPAWN_COMPLIANCE_MODE: "nist-800-171"
  SPAWN_COMPLIANCE_STRICT_MODE: "true"

test:
  script:
    - spawn launch --instance-type t3.micro --ttl 30m
    - # Run tests on instance
    - spawn terminate
```

## Troubleshooting

### Launch Fails with Compliance Violations

**Problem**: Launch fails immediately with compliance error messages.

**Solution**: Check the error message for the specific control ID and violation. Common issues:
- Missing security groups (SC-07)
- Attempting to disable encryption (SC-28)
- Invalid network configuration

### Warnings About Shared Infrastructure

**Problem**: Warnings about using shared infrastructure with compliance mode.

**Solution**: These are informational. Options:
1. Acknowledge and continue (acceptable for NIST 800-171)
2. Deploy self-hosted infrastructure (recommended for production)
3. Disable warnings: `allow_shared_infrastructure: true` in config

### Validation Shows Non-Compliant Instances

**Problem**: `spawn validate` reports instances as non-compliant.

**Solution**: Instances launched before enabling compliance mode won't be compliant. Options:
1. Terminate and relaunch with `--nist-800-171`
2. Document as legacy instances with waiver/exception
3. Enable default EBS encryption in AWS account settings

## Next Steps

- **Self-Hosted Infrastructure**: See [Self-Hosted Infrastructure Guide](../how-to/self-hosted-infrastructure.md)
- **NIST 800-53 Baselines**: For FedRAMP requirements, see [NIST 800-53 Baselines Guide](./nist-800-53-baselines.md)
- **Control Matrix**: Full control mapping in [Control Matrix](./control-matrix.md)
- **Audit Evidence**: Generating audit evidence in [Audit Evidence Guide](./audit-evidence.md)

## References

- [NIST SP 800-171 Rev 3](https://csrc.nist.gov/publications/detail/sp/800-171/rev-3/final)
- [AWS Compliance Programs](https://aws.amazon.com/compliance/programs/)
- [spawn Security Architecture](../architecture/security.md)
