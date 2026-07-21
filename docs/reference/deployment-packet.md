---
description: "Everything a cloud/security administrator needs to approve spore.host for a researcher on an institution-managed AWS account — in one page."
---

# Deployment packet (for AWS administrators)

A researcher on your team wants to use **spore.host** to launch short-lived,
self-terminating EC2 instances in your AWS account. This page is the one thing
they need to hand you: what it is, exactly what permissions it needs and why,
what it creates, what it audits, what (if anything) leaves the account, and how to
remove it.

**One-line summary:** spore.host is a set of CLI tools that run on the
researcher's machine with *their* AWS credentials and launch EC2 in *your*
account. It never receives your credentials or workload data. Its whole purpose is
to bound cost — every instance gets a hard termination deadline. See
[Security, credentials & data flow](/architecture) for the full trust map and
[Security Overview](https://github.com/spore-host/spore-host/blob/main/SECURITY.md)
for the CISO-oriented assessment.

## 1. The least-privilege policy to grant

The complete, verified baseline policy is on the
**[IAM Permissions](/reference/iam-permissions)** page — apply that policy to the
IAM role/user the researcher authenticates as. It grants:

- **EC2** describe + `RunInstances` + tag + start/stop/terminate (the last three
  **conditioned on `spawn:managed=true`**, so it can only act on instances spawn
  created — never anything else in the account).
- **EC2** `ImportKeyPair`, `CreateSecurityGroup`, `AuthorizeSecurityGroupIngress`
  (to install the user's SSH key and open SSH to the instance).
- **IAM** create-role / instance-profile / `PassRole` **scoped to `spored*` names
  only** — so it cannot create or touch any role that isn't named `spored*`.

Optional feature permissions (Spot, Route 53 DNS, FSx) are listed separately on
that page and are only needed if the researcher uses those flags.

## 2. What spawn creates in your account

| Resource | When | Notes |
|----------|------|-------|
| `spored-instance-role` + `spored-instance-profile` | Once per account, on first launch | Scoped so the instance can read its own tags and stop/terminate itself only |
| EC2 instances | Per `spawn launch` | Tagged `spawn:managed=true`, `spawn:ttl`, `spawn:created-by=spawn`, etc. |
| A security group | Per launch (if none supplied) | Opens SSH (22) to the launching user; you can supply your own instead |
| An imported EC2 key pair | Per user | The researcher's **public** key only |

The `spored` role/profile is the only standing resource; everything else is
ephemeral and self-terminates.

## 3. What it audits

Every action spawn takes is a normal AWS API call by the researcher's identity,
so it all lands in **CloudTrail** under their principal: `RunInstances`,
`TerminateInstances`, `CreateRole`/`CreateInstanceProfile` (first launch),
`ImportKeyPair`, `CreateSecurityGroup`, and the tagging calls. You can build a
Cost Allocation report from the `spawn:*` tags. See the
[Security Overview](https://github.com/spore-host/spore-host/blob/main/SECURITY.md#5-audit-and-compliance)
for example CloudTrail queries.

## 4. What leaves the account

**By default: nothing.** The CLI and the in-instance `spored` daemon run entirely
within your account. Optional hosted integrations (Slack/Teams notifications, DNS
subdomains) call out to spore.host's own AWS account over HTTPS and receive
**selected metadata only** (instance id, state, tags) — never credentials or
workload data — and only if the researcher enables them. Those can also be
[self-hosted](/guides/self-hosting) so nothing leaves the account at all. See the
["guarantee by deployment mode" table](https://github.com/spore-host/spore-host/blob/main/SECURITY.md#lifecycle-guarantee-by-deployment-mode).

## 5. Cost-safety posture (why this reduces risk)

spore.host exists to prevent forgotten instances. Every launch has a **TTL** — a
hard deadline enforced from inside the instance by `spored`, independent of the
researcher's laptop. You can additionally require budgets/SCPs as usual; spore.host
complements them. See [Costs & safety guarantees](/safety).

## 6. Predeployment validation

Before (and after) you grant the policy, the researcher can run:

```sh
spawn doctor
```

It performs read-only checks and reports exactly which permissions are present or
missing — so you can confirm the grant is correct without launching anything. If
`spawn doctor` reports all-clear, they're ready.

## 7. Removal / cleanup

To fully remove spore.host's footprint:

```sh
# terminate any managed instances
spawn terminate --all            # or: aws ec2 terminate-instances --instance-ids …
                                 #     (filter on tag:spawn:managed=true)

# then delete the standing IAM role + instance profile
aws iam remove-role-from-instance-profile --instance-profile-name spored-instance-profile --role-name spored-instance-role
aws iam delete-instance-profile --instance-profile-name spored-instance-profile
aws iam delete-role --role-name spored-instance-role
```

Detach the least-privilege policy from the researcher's identity to revoke access.
`spawn orphans` / `spawn cleanup` (run by the researcher) find and remove any
stragglers.

---

**Questions?** The [Security Overview](https://github.com/spore-host/spore-host/blob/main/SECURITY.md)
answers most security-team questions; for anything else, reach the maintainers via
[Discord](https://discord.gg/2deGRFCW) or a
[private security advisory](https://github.com/spore-host/spore-host/security/advisories/new).
