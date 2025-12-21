# Security Overview for mycelium (truffle + spawn)

**Audience:** CISOs, Cloud Administrators, Security Engineers

**Purpose:** Comprehensive security assessment and operational guidance for deploying mycelium tools in enterprise AWS environments.

---

## Executive Summary

**mycelium** is a command-line toolset for AWS EC2 instance discovery and ephemeral compute provisioning. It consists of two components:

- **truffle**: EC2 instance type search tool (read-only)
- **spawn**: EC2 instance launcher with auto-termination (read/write)

**Security Model:** Least-privilege IAM permissions, explicit resource tagging, and automatic cleanup of ephemeral resources.

**Risk Profile:** Medium - requires EC2 launch permissions and limited IAM role creation

**Compliance:** Supports AWS best practices, CloudTrail auditing, and resource tagging for cost allocation

---

## 1. Architecture Overview

### truffle (Read-Only Tool)

**Purpose:** Search and compare EC2 instance types across regions

**AWS Services Used:**
- EC2 DescribeInstanceTypes (read-only)
- EC2 DescribeRegions (read-only)
- EC2 DescribeAvailabilityZones (read-only)

**Security Posture:** Zero write permissions, no resource creation

**Data Access:** Only AWS service metadata (pricing, specifications)

**Risk Assessment:** **LOW** - Cannot modify infrastructure

---

### spawn (Instance Launcher)

**Purpose:** Launch ephemeral EC2 instances with automatic termination

**AWS Services Used:**
- **EC2:** Launch, terminate, and query instances
- **IAM:** Create service role for spawnd agent (once per account)
- **SSM Parameter Store:** Query latest Amazon Linux AMI IDs
- **S3:** Download spawnd agent binary (public read, SHA256-verified)

**Security Posture:** Requires EC2 launch permissions and limited IAM role creation

**Resource Lifecycle:** Automatic cleanup via TTL and idle detection

**Risk Assessment:** **MEDIUM** - Can launch instances (cost implications)

---

## 2. IAM Permissions Required

### For truffle (Read-Only)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstanceTypes",
        "ec2:DescribeRegions",
        "ec2:DescribeAvailabilityZones"
      ],
      "Resource": "*"
    }
  ]
}
```

**Compatible with:** ReadOnlyAccess, ViewOnlyAccess, PowerUserAccess

---

### For spawn (Minimum Required)

See `spawn/IAM_PERMISSIONS.md` for complete policy.

**Summary:**
- **EC2:** Launch, terminate, describe instances (standard compute access)
- **IAM:** Create `spawnd-instance-role` and `spawnd-instance-profile` (one-time setup)
- **SSM:** Read `/aws/service/ami-amazon-linux-latest/*` (AMI auto-detection)

**Compatible with:** PowerUserAccess + spawn-specific IAM permissions (see below)

**NOT Compatible with:** PowerUserAccess alone (missing IAM permissions)

---

### PowerUser + spawn IAM Policy

For organizations using AWS `PowerUser` managed policy, add this supplementary policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "SpawnIAMRoleManagement",
      "Effect": "Allow",
      "Action": [
        "iam:CreateRole",
        "iam:GetRole",
        "iam:PutRolePolicy",
        "iam:CreateInstanceProfile",
        "iam:GetInstanceProfile",
        "iam:AddRoleToInstanceProfile",
        "iam:PassRole"
      ],
      "Resource": [
        "arn:aws:iam::*:role/spawnd-instance-role",
        "arn:aws:iam::*:instance-profile/spawnd-instance-profile"
      ]
    },
    {
      "Sid": "SpawnIAMTagging",
      "Effect": "Allow",
      "Action": [
        "iam:TagRole",
        "iam:TagInstanceProfile"
      ],
      "Resource": [
        "arn:aws:iam::*:role/spawnd-instance-role",
        "arn:aws:iam::*:instance-profile/spawnd-instance-profile"
      ]
    }
  ]
}
```

**Why:** PowerUser excludes all IAM permissions. This grants minimal IAM access scoped to spawn-specific resources only.

---

## 3. IAM Role Created by spawn

### spawnd-instance-role

**Purpose:** Allows spawnd agent (running on EC2 instances) to self-manage based on tags

**Trust Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

**Permissions Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeTags",
        "ec2:DescribeInstances"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:TerminateInstances",
        "ec2:StopInstances"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ec2:ResourceTag/spawn:managed": "true"
        }
      }
    }
  ]
}
```

**Security Controls:**
- ✅ Can only terminate/stop instances tagged `spawn:managed=true`
- ✅ Cannot terminate instances launched by other tools
- ✅ Cannot modify IAM policies
- ✅ Cannot access S3, databases, or other AWS services
- ✅ Scoped to EC2 instance lifecycle only

---

## 4. Security Features

### Automatic Resource Cleanup

**TTL (Time-To-Live):**
- User specifies maximum instance lifetime (e.g., `--ttl 8h`)
- spawnd agent monitors uptime
- Terminates instance when TTL expires
- **Benefit:** Prevents forgotten instances running indefinitely

**Idle Detection:**
- User specifies idle timeout (e.g., `--idle-timeout 30m`)
- spawnd monitors SSH sessions and CPU utilization
- Terminates or hibernates when idle threshold reached
- **Benefit:** Reduces cost for intermittent workloads

**Laptop Independence:**
- spawnd runs as systemd service on the instance (not laptop)
- Auto-termination works even if laptop is off or disconnected
- **Benefit:** Prevents orphaned resources from VPN disconnections

---

### Resource Tagging

All instances launched by spawn are tagged:

```
spawn:managed = true
spawn:root = true
spawn:created-by = spawn
spawn:version = 0.1.0
spawn:ttl = <duration>           (if specified)
spawn:idle-timeout = <duration>  (if specified)
```

**Benefits:**
- Cost allocation and chargeback
- Automated cleanup policies (AWS Config, Lambda)
- Compliance reporting (which instances have TTL?)
- Security auditing (CloudTrail queries by tag)

---

### Binary Distribution Security

**Problem:** How to securely distribute spawnd binary to instances?

**Solution:** Public S3 with SHA256 verification (industry standard)

**Implementation:**
1. spawnd binaries stored in public S3 buckets (one per region)
2. Each binary has corresponding `.sha256` checksum file
3. Instance user-data downloads both binary and checksum
4. SHA256 verified before execution: `sha256sum --check spawnd-linux-amd64.sha256`
5. Installation fails if checksum mismatch

**Why Public S3?**
- Same model as apt/yum/pip repositories
- No AWS credentials in user-data (reduces attack surface)
- Fast downloads (regional buckets, <20ms latency)
- Tamper detection via cryptographic checksums

**Threat Model:**
- ❌ **Compromised S3 Bucket:** Attacker cannot upload malicious binary without SHA256 key
- ❌ **Man-in-the-Middle:** HTTPS + SHA256 verification prevents tampering
- ✅ **Authorized Updates:** Only spawn maintainers can update binaries (S3 bucket policy)

**S3 Bucket Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadSpawndBinaries",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::spawn-binaries-*/spawnd-*"
    }
  ]
}
```

---

### SSH Key Management

**Fingerprint-Based Key Reuse:**
- spawn calculates MD5 fingerprint of local SSH public key
- Searches AWS EC2 for existing key pair with matching fingerprint
- Reuses existing key if found (no duplicate uploads)
- Only uploads if no matching key exists

**Benefits:**
- No duplicate keys cluttering AWS account
- Consistent key usage across multiple launches
- User can pre-import keys with custom names

**Security:**
- Only public keys handled (private keys never leave laptop)
- Keys stored in AWS EC2 KeyPairs (standard AWS service)
- No custom key storage or management

---

## 5. Audit and Compliance

### CloudTrail Events

All spawn operations generate CloudTrail events:

**IAM Role Creation:**
```
CreateRole (spawnd-instance-role)
PutRolePolicy (spawnd-policy)
CreateInstanceProfile (spawnd-instance-profile)
AddRoleToInstanceProfile
```

**Instance Launch:**
```
RunInstances
  - Tags: spawn:managed=true, spawn:ttl=8h
  - IamInstanceProfile: spawnd-instance-profile
  - UserData: <base64-encoded script>
```

**Instance Termination (by spawnd):**
```
TerminateInstances
  - InitiatedBy: spawnd (via instance profile role)
  - Reason: TTL expired / idle timeout
```

**Query Examples:**

Find all spawn-launched instances:
```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance \
  --query 'Events[?contains(CloudTrailEvent, `spawn:managed`)]'
```

Find instances terminated by spawnd:
```bash
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=TerminateInstances \
  --query 'Events[?contains(CloudTrailEvent, `spawnd-instance-role`)]'
```

---

### Cost Control

**Built-In Mechanisms:**
1. **TTL Enforcement:** Hard limit on instance runtime
2. **Idle Detection:** Automatic termination of unused instances
3. **Resource Tagging:** Cost allocation via `spawn:*` tags
4. **Spot Instance Support:** `--spot` flag for 70% savings

**Recommended Controls:**
1. **AWS Budgets:** Alert on `spawn:managed=true` tag spend
2. **Service Control Policies:** Limit instance types per OU
3. **IAM Conditions:** Require `--ttl` flag via policy conditions
4. **Cost Anomaly Detection:** Alert on unusual spawn usage patterns

**Example Budget:**
```bash
aws budgets create-budget \
  --account-id 123456789012 \
  --budget file://spawn-budget.json
```

```json
{
  "BudgetName": "spawn-instances-monthly",
  "BudgetLimit": {
    "Amount": "1000",
    "Unit": "USD"
  },
  "TimeUnit": "MONTHLY",
  "BudgetType": "COST",
  "CostFilters": {
    "TagKeyValue": ["user:spawn:managed$true"]
  }
}
```

---

### Compliance Considerations

**GDPR / Data Residency:**
- spawn respects regional boundaries (no cross-region data transfer)
- Users control region via `--region` flag
- AMI selection automatic per region

**PCI-DSS / HIPAA:**
- spawn does not handle sensitive data
- Instances launched in user's VPC (user controls network security)
- Encryption at rest supported (EBS encryption via instance type requirements)

**SOC 2 / ISO 27001:**
- Audit trail via CloudTrail
- Resource tagging for inventory management
- Automatic cleanup reduces security surface area

---

## 6. Risk Assessment

### Identified Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| Excessive instance launches | Medium | AWS Budgets, SCPs, `--ttl` requirement |
| IAM role privilege escalation | Low | Role scoped to `spawn:managed=true` only |
| Binary tampering | Low | SHA256 verification, HTTPS |
| Forgotten instances | Low | TTL enforcement, idle detection |
| Insider threat (malicious user) | Medium | CloudTrail auditing, IAM least privilege |
| Credential exposure | Medium | No credentials in user-data, IMDSv2 supported |

### Mitigations Summary

✅ **Least Privilege IAM:** Users only get permissions they need
✅ **Resource Tagging:** All resources identifiable and auditable
✅ **Automatic Cleanup:** TTL/idle detection prevents resource leaks
✅ **Audit Trail:** CloudTrail captures all operations
✅ **Cost Controls:** Budgets, SCPs, Spot instances
✅ **Binary Integrity:** SHA256 verification
✅ **Network Security:** User controls VPC, security groups, subnets

---

## 7. Deployment Recommendations

### For Small Teams (< 50 users)

1. **IAM Users with spawn policy** (see `IAM_PERMISSIONS.md`)
2. **AWS Budget alert** at $500/month
3. **CloudTrail enabled** (standard AWS best practice)
4. **Require `--ttl` via documentation** (not enforced)

### For Enterprises (> 50 users)

1. **IAM Groups:**
   - `spawn-power-users`: Full spawn access
   - `spawn-basic-users`: spawn + instance type restrictions via SCP

2. **Service Control Policies:**
   ```json
   {
     "Effect": "Deny",
     "Action": "ec2:RunInstances",
     "Resource": "arn:aws:ec2:*:*:instance/*",
     "Condition": {
       "StringNotLike": {
         "ec2:InstanceType": [
           "t3.*",
           "t4g.*",
           "m7i.large",
           "m7i.xlarge"
         ]
       },
       "StringEquals": {
         "aws:PrincipalTag/spawn-user": "true"
       }
     }
   }
   ```

3. **Mandatory TTL via IAM Policy:**
   ```json
   {
     "Effect": "Deny",
     "Action": "ec2:RunInstances",
     "Resource": "arn:aws:ec2:*:*:instance/*",
     "Condition": {
       "StringNotEquals": {
         "aws:RequestTag/spawn:ttl": "*"
       }
     }
   }
   ```
   *(Note: Requires spawn to always set TTL tag)*

4. **Cost Allocation Tags:**
   - Enable `spawn:*` tags as cost allocation tags
   - Monthly reports by team/project

5. **Security Hub Integration:**
   - Custom AWS Config rule: "Instances must have `spawn:ttl` tag"
   - Alert on non-compliant spawn instances

---

## 8. Incident Response

### Scenario: Unauthorized Instance Launch

**Detection:**
```bash
# Find all spawn instances
aws ec2 describe-instances \
  --filters "Name=tag:spawn:managed,Values=true" \
  --query 'Reservations[].Instances[].[InstanceId,LaunchTime,State.Name]'

# Check CloudTrail for who launched
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance
```

**Response:**
1. Identify user via CloudTrail `userIdentity`
2. Terminate instance: `aws ec2 terminate-instances --instance-ids i-xxxxx`
3. Review user IAM permissions
4. Check AWS Budget impact

---

### Scenario: spawnd Role Privilege Escalation Attempt

**Detection:** CloudWatch Logs Insights query:

```
fields @timestamp, errorCode, errorMessage
| filter eventSource = "ec2.amazonaws.com"
| filter userIdentity.principalId like /spawnd-instance-role/
| filter errorCode like /Unauthorized/
```

**Response:**
1. Identify instance: Extract instance ID from CloudTrail
2. Terminate instance immediately
3. Review spawnd role policy (should be read-only except for self-termination)
4. Check for IAM policy modifications

---

## 9. Validation and Testing

### Pre-Deployment Checklist

- [ ] Run `./scripts/validate-permissions.sh <aws-profile>` for each user
- [ ] Create test AWS Budget for spawn resources
- [ ] Enable CloudTrail in all regions
- [ ] Test IAM role creation (first spawn launch)
- [ ] Verify TTL enforcement (launch instance with `--ttl 5m`)
- [ ] Verify idle detection (launch instance with `--idle-timeout 10m`)
- [ ] Test Spot instances (`--spot` flag)
- [ ] Review CloudTrail events for spawn operations

### Security Testing

```bash
# Test 1: Verify spawnd cannot terminate non-spawn instances
aws ec2 run-instances --image-id ami-xxxxx --instance-type t3.micro
# (manually launched, no spawn:managed tag)
# spawnd should fail to terminate this

# Test 2: Verify TTL enforcement
echo '[{"instance_type":"t3.micro","region":"us-east-1"}]' | \
  spawn launch --ttl 5m
# Wait 6 minutes, verify instance terminated

# Test 3: Verify SHA256 verification
# Corrupt spawnd binary on S3, verify instance launch fails

# Test 4: Verify CloudTrail logging
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=EventName,AttributeValue=RunInstances
```

---

## 10. Frequently Asked Questions (Security)

### Q: Can spawn access my existing EC2 instances?

**A:** No. spawn can only launch new instances and terminate instances tagged `spawn:managed=true`. It cannot access, modify, or terminate instances launched by other tools.

---

### Q: What data does spawn send to external services?

**A:** None. spawn only communicates with:
- AWS APIs (EC2, IAM, SSM)
- S3 buckets for spawnd binary download (public, read-only)

No telemetry, analytics, or user data leaves your AWS account.

---

### Q: Can spawnd access my S3 buckets or databases?

**A:** No. The `spawnd-instance-role` only has permissions for:
- Reading its own EC2 tags
- Terminating/stopping itself

It cannot access S3, RDS, DynamoDB, or any other AWS services.

---

### Q: What if someone modifies the spawnd binary on S3?

**A:** The instance will fail to launch. User-data verifies SHA256 checksum before executing spawnd. If the binary is tampered, the checksum won't match and installation aborts.

---

### Q: Can users bypass TTL limits?

**A:** Users cannot extend TTL after launch. However, they could:
- Manually terminate the instance and launch a new one (new TTL starts)
- SSH in and kill the spawnd service (instance won't auto-terminate)

**Mitigation:** CloudWatch Events rule to detect spawnd service failures.

---

### Q: How do I audit spawn usage?

**A:** Use CloudTrail:

```bash
# All spawn launches (last 7 days)
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::EC2::Instance \
  --max-results 1000 \
  --query 'Events[?contains(CloudTrailEvent, `spawn:managed`)].{Time:EventTime,User:Username,Instance:Resources[0].ResourceName}'

# Cost by user (requires Cost Allocation Tags)
aws ce get-cost-and-usage \
  --time-period Start=2025-12-01,End=2025-12-31 \
  --granularity MONTHLY \
  --group-by Type=TAG,Key=spawn:created-by \
  --metrics BlendedCost
```

---

## 11. Contact and Support

**Security Issues:** Report to project maintainers (see CONTRIBUTING.md)

**Documentation:**
- Full IAM policy: `spawn/IAM_PERMISSIONS.md`
- User guide: `spawn/README.md`
- Validation script: `scripts/validate-permissions.sh`

**References:**
- [AWS IAM Best Practices](https://docs.aws.amazon.com/IAM/latest/UserGuide/best-practices.html)
- [AWS Well-Architected Security Pillar](https://docs.aws.amazon.com/wellarchitected/latest/security-pillar/welcome.html)
- [CloudTrail Event Reference](https://docs.aws.amazon.com/awscloudtrail/latest/userguide/cloudtrail-event-reference.html)

---

**Document Version:** 1.0
**Last Updated:** December 21, 2025
**Next Review:** January 21, 2026
