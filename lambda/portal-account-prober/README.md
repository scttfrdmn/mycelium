# portal-account-prober

The scheduled caller `accountlifecycle.ApplyProbes` was written for (spawn#457).

Every run: scan the `spore-portal-accounts` registry, assume each account's
`spore-portal-onboard` role, count `spawn:managed=true` instances across every
region, hand the results to `ApplyProbes`, and persist only the rows it says
changed. Runs in the infra account (966362334030) on an EventBridge schedule.

It exists so spore.host can eventually conclude "this account is gone" and expire
the Route53 A-records that would otherwise outlive their instances and resolve to a
stranger's IP. The state machine, the four statuses, and the `DNSExpiryEligible`
gate all live in [`../accountlifecycle`](../accountlifecycle/) — read that first.
This module is only the observer.

## Why a separate Lambda

Two existing functions already do something adjacent, and neither could take this on:

- **spawn's `ttl-reaper`** assumes into customer accounts every 10 minutes — but a
  *different* role: `spawn-ttl-reaper-ec2`, created by a manual CloudFormation
  deploy and listed in `REAPER_ROLE_ARNS`. Onboarding creates
  `spore-portal-onboard`, which trusts only the phone-home Lambda. So the reaper
  cannot assume the roles the registry knows about; pointing it at the registry
  would mean probing accounts it has no credentials for and recording their
  unreachability as fact.
- **`portal-phone-home`** already holds the trust relationship — but it is
  internet-facing under a Function URL. Granting it `sts:AssumeRole` into every
  customer account would put that capability one handler bug from the public edge.

This function has no URL and no public trigger. One EventBridge rule is its only
invoker.

## What a probe observes

`probeAccount` returns one `ProbeResult` per account, and the whole design turns on
getting its three fields right — `ApplyProbes` guards its *transitions* carefully,
but it can only be as correct as the observation it is handed.

| Field | Set when |
|---|---|
| `Reachable` | assume-role + `DescribeInstances` succeeded in **at least one** region |
| `AssumeRoleDenied` | assume-role was refused (`AccessDenied` / `UnauthorizedOperation`) |
| `EmptinessUnproven` | reached *some* regions but not all |

Three details that are not obvious:

**A denied assume-role is one observation, not eleven.** Credentials are per
account, not per region, so the first denial returns immediately rather than
retrying the remaining ten regions and reporting a pile of identical failures.

**`stopped` instances count as live.** `countManaged` filters
`pending|running|stopping|stopped`. An idle-stopped spore is not evidence the
account is unused — it is evidence of a researcher between jobs, which is exactly
who the N-day dormancy clock exists to protect.

**A pagination failure discards the partial count** (`return 0, err`). A truncated
count reads as "emptier than reality", and emptier is the direction that
deprovisions. Covered by `TestProbeAccount_MidPaginationFailureIsPartialCoverage`,
which needs a fake that fails on page *two* — failing on page one cannot tell the
two behaviours apart.

## The rollout hazard this is built around

STS returns `AccessDenied` both when a role has been **deleted** — the uninstall
signal this whole design reads — and when the role exists but its trust policy
doesn't name our principal. That ambiguity is deliberate on AWS's part (it avoids
leaking role existence), and the error code cannot discriminate.

Which matters enormously here, because **every `spore-portal-onboard` trust policy
written before this Lambda existed names only the phone-home role.** They will all
refuse the prober. Acting on the first reading when the second is true would march
the entire pre-existing customer base to `unreachable` over the prober's first six
runs.

The correlated-failure guard cannot catch it: that guard fires only when *every*
probe fails, and this affects a subset — every legacy account while every new one
succeeds. `accountlifecycle` resolves it with the registry's own `LastSeenAt`: **a
denial is evidence only for an account that has succeeded at least once**, because
such an account's trust policy provably admitted us, so a refusal now is a change on
the customer's side. With no baseline we conclude nothing.

`TestRun_PreExistingAccountsSurviveTheProberRollout` runs three baseline-less
accounts denying us for 3×K runs *alongside a healthy one* — so the
correlated-failure guard is demonstrably not what spares them — and asserts no
writes and no status change. `TestRun_UninstallAfterBaselineIsDetected` is its
mandatory companion: without it, a prober that never concludes anything would pass.

## The session policy

`spore-portal-onboard` is a **launch** role: `RunInstances`, `TerminateInstances`,
`iam:PassRole`. It has to be — that is what the portal needs. And a trust policy
governs *who may assume*, not *what they may then do*. So naming the prober as a
trusted principal hands it the full launch capability, to make one read call with.

Every assume-role therefore attaches an inline STS **session policy** allowing only
`ec2:DescribeInstances`. Effective permissions are the intersection, so these
credentials cannot launch or terminate anything even if this function is
compromised. That self-limit is what makes it honest to ask customers for the grant
instead of shipping a second read-only role into their account, and
`TestSessionPolicy_IsValidAndReadOnly` asserts the document itself — a malformed
policy would fail every assume at runtime, and a widened one would silently remove
the cap.

`Retrieve()` is also called eagerly, so a refusal surfaces as an assume-role error
rather than later inside `DescribeInstances`, where it would be indistinguishable
from an EC2-level denial.

## Configuration

| Env var | Default | Notes |
|---|---|---|
| `ACCOUNTS_TABLE` | `spore-portal-accounts` | the registry |
| `PROBER_REGIONS` | 11 regions (the reaper's set) | **correctness knob, not a cost knob** |
| `PROBER_FAILURES_BEFORE_UNREACHABLE` | 6 | K, counted in **runs** |
| `PROBER_DORMANT_AFTER` | `720h` | N |
| `PROBER_DRY_RUN` | unset | `true` = decide and log, persist nothing |

**`PROBER_REGIONS` is a correctness input.** Emptiness is established by finding
nothing across the whole set, so a region missing from here is a region whose
instances are invisible to dormancy. Shrinking it to save API calls trades directly
against the risk of calling a busy account dormant.

**K is counted in runs, not elapsed time**, so the EventBridge rate is a policy
input: K=6 is six hours at `rate(1 hour)` and one hour at `rate(10 minutes)`.

## The alarm

`ApplyProbes` changing nothing when every probe fails is correct but *silently*
correct — a run that concludes nothing because our own credentials broke looks
exactly like a run with nothing to do. So the handler sets `Summary.Refused` and
logs `REFUSING to conclude anything`, and the Tofu module puts a log metric filter
and alarm on that string. Alarm on it, not on `Errors`. Without it the prober can be
completely broken for weeks while reporting clean runs.

## Tests

```bash
go test ./...
```

No AWS. `probeAccount` is driven through a `regionalEC2` factory seam
(`regionScript` scripts per-region counts and errors); the run loop is driven
through a `fakeRegistry` that records **every** write, because the sharpest
assertions in this package are about writes that must *not* happen.

`probeAccount`, `isDenied`, and `countManaged` are at 100%. Mutation-verified:
neutering the denial classification, the reached-any accounting, the
partial-coverage flag, the refusal guard, or the pagination-error discard each fails
a test.

## Build & deploy

```bash
make build            # linux/arm64 bootstrap + function.zip
make upload           # → s3://spawn-binaries-us-east-1/portal-account-prober/
```

Infrastructure is OpenTofu: [`infra/tofu/portal-account-prober/`](../../infra/tofu/portal-account-prober/).

One manual step has no automation: the module's `role_arn` output must be pasted
into `portal-onboarding-role.yaml`'s `ProberLambdaRoleArn` parameter and the
template re-uploaded to the website bucket, or newly onboarded accounts' trust
policies won't admit the prober either.
