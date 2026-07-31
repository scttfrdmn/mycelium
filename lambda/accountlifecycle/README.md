# accountlifecycle

The BYOA account registry and its **deprovisioning state machine** (spawn#457).

Shared by the two Lambdas that touch the registry table `spore-portal-accounts`:
[`portal-phone-home`](../portal-phone-home/) writes registrations,
[`portal-account-prober`](../portal-account-prober/) writes lifecycle transitions.

It is its own Go module for one reason: so `ApplyProbes` has exactly **one** copy.
Its refusals are mutation-verified, and a second copy would drift out from under
those tests.

## The problem

Onboarding was one-way. The registry exposed `PutAccount` and `GetAccount` and
nothing else, so there was no state in which spore.host could conclude "this account
is gone" — and everything left behind was therefore permanent.

Most of that residue is harmless. One piece is not: a Route53 A-record under
`{base36}.spore.host` outlives the instance it named, and the public IP it points at
returns to the general EC2 pool. Once AWS reassigns that address, the record resolves
to a stranger's instance. That inverts the usual instinct — the security consequence
argues for expiring DNS records **sooner**, not keeping them around just in case.

Note what we *cannot* expire: `spawn-ttl-reaper-ec2` is created in the **customer's**
account. We hold only the trust relationship. Only they can delete it, and an IAM
role with no caller is inert and free, so no cost pressure ever forces the issue.

## The signal

An assume-role probe distinguishes exactly the states that matter:

- assume **succeeds**, zero managed instances for N days → *dormant but reachable*
- assume **fails** while other accounts succeed → *the customer uninstalled*

Role deletion is the natural uninstall gesture. No new API, and no customer action
beyond what they would already do.

## The four states

Each state exists to answer one question: **is it safe to delete this account's DNS
records?**

| Status | Meaning | DNS expiry eligible |
|---|---|---|
| `active` | assume-role works (also: any legacy row with no `status`) | no — in use |
| `unreachable` | assume-role failed K consecutive runs *while others succeeded* | **no** — see trap 2 |
| `dormant` | assume-role **works**, zero managed instances for N | yes — emptiness proven |
| `offboarded` | a human said so | yes — intent stated |

`DNSExpiryEligible()` is the single gate. Unknown/future status values return false,
so a typo can never authorize a deletion.

## Three doors into a false deprovision, and how each is shut

**1. Correlated failure is indistinguishable from mass uninstall.** Role ARNs can
embed a CloudFormation-generated physical ID; recreating a stack changes the suffix
and breaks **every** customer's trust policy simultaneously. So `ApplyProbes`
**changes nothing when every probe fails** — an observation explainable by our own
breakage is not evidence about the customer. Same instinct as the DNS sweep's
existing refusal to delete against a partial live set.

The sharp edge is the single-account deployment, where "all probes failed" and "this
one account failed" are the same observation. The guard resolves it as uninformative,
by design (`TestApplyProbes_SingleAccountFailureIsUninformative`).

**2. A *subset* can deny us for reasons that are ours, not theirs.** STS returns
`AccessDenied` both when a role has been deleted — the uninstall signal — and when
the role exists but its trust policy doesn't name our principal. AWS makes them
identical deliberately, so as not to leak role existence, and the error code cannot
discriminate. Trap 1's guard structurally cannot help: it fires only when *every*
probe fails, and a trust-policy gap affects a subset.

The discriminator is in the registry itself: **`LastSeenAt`**. A denial is evidence
only for an account that has succeeded at least once — such an account's trust policy
provably admitted us, so a refusal *now* is a change on the customer's side. With no
baseline we conclude nothing. This is what makes the prober's own rollout safe: every
role onboarded before the prober existed will deny it, and is left strictly alone.

**3. Emptiness can be concluded from a partial look.** Dormancy means "reachable AND
empty", and emptiness is proven by finding nothing *everywhere* — so zero-of-a-partial-set
is not zero. A probe that reached some regions but not all sets `EmptinessUnproven`,
and the dormancy evaluation is skipped.

Liveness is still stamped, because reachability *was* proven. `LastInstanceAt` is
deliberately left alone rather than re-stamped: re-stamping would reset the N-day
clock, so one chronically failing region would block dormancy forever. Skipping
merely defers it.

## Trap 2's mandatory converse

A state machine that concludes nothing is trivially safe and completely useless.
`TestApplyProbes_DeniedAfterPriorSuccessIsEvidence` is therefore not optional — it is
the test that proves an uninstall is still *detected*. Without it, every guard above
could be tightened until the design reads no signal at all and the suite would still
pass.

## Policy

`DefaultLifecyclePolicy()` is **K=6** consecutive failed runs and **N=30 days**
reachable-and-empty (long enough that a researcher between jobs is not deprovisioned
mid-project). Both are fields on `LifecyclePolicy`.

**K is counted in runs, not elapsed time**, so the caller's schedule is a policy
input: six hours at the prober's `rate(1 hour)`, one hour at the reaper's
`rate(10 minutes)`.

Other properties worth knowing:

- **`unreachable` stops counting into the caller's `Errors`.** That field means
  "investigate this" and is worthless once it holds a permanent expected failure — a
  forever-failing assume-role trains the operator to ignore the count.
- **A never-used account is not instantly dormant.** Absent evidence is not evidence
  of absence, so the N-day clock is seeded from first observation.
- **Recovery is automatic, revival is not.** A `dormant` or `unreachable` account that
  runs a spore again returns to `active`. An `offboarded` one never does — that status
  was a human decision and only a human (a re-onboard) undoes it.
- **An unparseable timestamp never triggers a transition.** Acting on a value we
  cannot read is how a healthy account gets deprovisioned by a formatting bug.

## Trap 2 vs trap 1: which state does `unreachable` get?

Both trap-1 and trap-2 refusals leave the row untouched, but for different reasons,
and the distinction matters when reading logs: trap 1 means *we* are broken (alarm on
it — the prober's `REFUSING to conclude anything`), trap 2 means *this account* has
never let us in (expected, and permanent until it re-onboards).

## Schema compatibility

Every lifecycle field is `omitempty`, so rows written before this change unmarshal to
the zero value. `AccountStatus()` maps `""` → `active`. **No backfill migration.**
Read the status through that method, never off `.Status` directly, or legacy rows look
like an unknown state.

## Persistence

- `ListAccounts` — paginated Scan. One row per onboarded account (tens, not
  millions). Paginated anyway because a truncated Scan would look exactly like "these
  accounts no longer exist", the precise false conclusion this work exists to prevent.
- `UpdateLifecycle` — `UpdateItem`, writing **only** the lifecycle attributes. Not a
  Put, in two directions: a Put would clobber a concurrent re-onboard's fresh
  ExternalId/roleArn with the stale copy a probe run happened to read, and because
  UpdateItem upserts, it would also resurrect a row deliberately deleted between the
  Scan and the write. Empty timestamps are omitted rather than written as `""`, since
  the dormancy math reads them back.
- `PutAccount` / `GetAccount` — registration, used by `portal-phone-home`.
- `Offboard` — the explicit human transition.
- **No `DeleteItem`.** The registry row costs nothing and is the audit trail;
  offboarding is a status transition, not a deletion.

## Tests

```bash
go test ./...
```

`ApplyProbes` and `DNSExpiryEligible` are pure and need no AWS. The persistence tests
run against substrate's in-memory DynamoDB (`substrate.StartTestServer`), not a real
table.

The state machine's guards are mutation-verified: neutering the correlated-failure
guard, the `LastSeenAt` baseline, the `EmptinessUnproven` deferral, the K threshold,
the N threshold, or either `offboarded` exclusion each fails a test. One redundancy is
documented in place — the never-used-account clock seed reaches the same result as the
unparseable-timestamp branch, and is kept because "never observed" and "observed but
unreadable" are different facts that happen to share a remedy.
