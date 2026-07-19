# IAM Permissions

spore.host tools act on EC2 with **your** AWS credentials (see [AWS Authentication](../guides/aws-auth.md) for how those are obtained). This page is the least-privilege IAM policy those credentials need. Attach it to the IAM role/user you authenticate as, and expand only as your usage grows (Spot, DNS, FSx below).

## Minimal policy

This is the complete set required for the core flow — **`spawn launch` → connect → manage → terminate**. It is verified against the actual AWS API calls spawn makes; a smaller policy will fail at launch (see the callout below).

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "EC2ReadAndLaunch",
      "Effect": "Allow",
      "Action": [
        "ec2:RunInstances",
        "ec2:DescribeInstances",
        "ec2:DescribeInstanceTypes",
        "ec2:DescribeInstanceTypeOfferings",
        "ec2:DescribeInstanceStatus",
        "ec2:DescribeImages",
        "ec2:DescribeSubnets",
        "ec2:DescribeVpcs",
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeKeyPairs",
        "ec2:DescribeAvailabilityZones",
        "ec2:DescribeSpotPriceHistory"
      ],
      "Resource": "*"
    },
    {
      "Sid": "EC2ProvisionSupport",
      "Effect": "Allow",
      "Action": [
        "ec2:ImportKeyPair",
        "ec2:CreateSecurityGroup",
        "ec2:AuthorizeSecurityGroupIngress",
        "ec2:CreateTags"
      ],
      "Resource": "*"
    },
    {
      "Sid": "InstanceProfileForSpored",
      "Effect": "Allow",
      "Action": [
        "iam:GetRole",
        "iam:CreateRole",
        "iam:PutRolePolicy",
        "iam:AttachRolePolicy",
        "iam:GetInstanceProfile",
        "iam:CreateInstanceProfile",
        "iam:AddRoleToInstanceProfile"
      ],
      "Resource": [
        "arn:aws:iam::*:role/spored*",
        "arn:aws:iam::*:instance-profile/spored*"
      ]
    },
    {
      "Sid": "PassSporedRoleToEC2",
      "Effect": "Allow",
      "Action": "iam:PassRole",
      "Resource": "arn:aws:iam::*:role/spored*",
      "Condition": { "StringEquals": { "iam:PassedToService": "ec2.amazonaws.com" } }
    },
    {
      "Sid": "EC2Manage",
      "Effect": "Allow",
      "Action": [
        "ec2:StartInstances",
        "ec2:StopInstances",
        "ec2:TerminateInstances",
        "ec2:DeleteTags",
        "ec2:DescribeTags",
        "ec2:ModifyInstanceAttribute"
      ],
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "ec2:ResourceTag/spawn:managed": "true"
        }
      }
    },
    {
      "Sid": "Identity",
      "Effect": "Allow",
      "Action": ["sts:GetCallerIdentity"],
      "Resource": "*"
    }
  ]
}
```

::: warning Why the extra statements are required
A launch does more than `RunInstances`. spawn **imports your SSH public key** (`ec2:ImportKeyPair`), may **create a security group** for the instance (`ec2:CreateSecurityGroup` / `AuthorizeSecurityGroupIngress`), and — so the in-instance `spored` daemon can manage the instance's lifecycle — **creates a `spored` IAM role + instance profile and passes it to EC2 at launch** (the `iam:*` on `spored*` + `iam:PassRole`). Omit these and `spawn launch` fails partway. The `iam:*` grants are scoped to `spored*` names so they can't touch other roles.
:::

::: tip Scope on tags
`EC2Manage` is conditioned on `spawn:managed=true`, so you can only start/stop/terminate instances spawn created — never unrelated ones. The read (`Describe*`) and launch statements can't be tag-scoped (the resources don't exist yet at describe/launch time).
:::

## Spot instances

Add these actions to request Spot capacity:

```json
{
  "Action": [
    "ec2:RequestSpotInstances",
    "ec2:CancelSpotInstanceRequests",
    "ec2:DescribeSpotInstanceRequests"
  ]
}
```

## DNS integration

If you're using `--dns` (Route 53 subdomain assignment):

```json
{
  "Action": [
    "route53:ChangeResourceRecordSets",
    "route53:ListResourceRecordSets",
    "route53:GetHostedZone"
  ],
  "Resource": "arn:aws:route53:::hostedzone/YOUR_ZONE_ID"
}
```

## FSx for Lustre

For shared filesystem integration (`spawn launch --fsx`):

```json
{
  "Action": [
    "fsx:CreateFileSystem",
    "fsx:DeleteFileSystem",
    "fsx:DescribeFileSystems",
    "fsx:CreateDataRepositoryTask",
    "cloudformation:CreateStack",
    "cloudformation:DeleteStack",
    "cloudformation:DescribeStacks"
  ]
}
```

## Service quotas

truffle checks instance quotas before suggesting instance types. Add:

```json
{
  "Action": [
    "servicequotas:GetServiceQuota",
    "servicequotas:ListServiceQuotas"
  ],
  "Resource": "*"
}
```

## Applying the policy

Save the minimal policy to `spore-host-policy.json`, then attach it to the role or user you authenticate as ([aws login](../guides/aws-auth.md) or a profile):

```sh
aws iam put-role-policy \
  --role-name your-spore-host-role \
  --policy-name spore-host \
  --policy-document file://spore-host-policy.json
```

Add the [Spot](#spot-instances), [DNS](#dns-integration), [FSx](#fsx-for-lustre), or [service-quota](#service-quotas) statements only if you use those features.

::: tip Prefer least privilege over `PowerUserAccess`
It's tempting to attach `PowerUserAccess` while getting started, but the policy above is small and complete — start with it. If you must broaden temporarily, tighten back to this policy before any shared or production use.
:::
