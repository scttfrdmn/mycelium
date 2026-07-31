# portal-phone-home

The spore.host portal's BYOA onboarding registrar, and the home of the
**account lifecycle state machine**.

Runs in the infra account (966362334030) — the control plane. One DynamoDB table,
`spore-portal-accounts`, keyed on `accountId`.

## Registration

When a user onboards their AWS account (via the `spawn onboard` CLI wizard or the
web CloudFormation quick-create), the newly created cross-account role phones home
here to register `{roleArn, externalId, region}`.

**Security — the SigV4-verified-principal model** (mirrors `dns-updater`,
spawn#173): the Function URL runs under `AuthType: AWS_IAM`, so every request that
reaches the handler has already passed SigV4 verification and carries the *verified*
caller account in `requestContext.authorizer.iam`. We trust that account, never
anything in the body. The body's `roleArn` must belong to the verified account or
we reject — so a caller can only ever register a role in its own account. No shared
secret, no allow-list, no spoofable claims. `validate()` is pure and holds this
invariant; `TestValidate_SecurityInvariant` covers it with no AWS.

## Account lifecycle (spawn#457)

### The problem

Onboarding was one-way. The registry exposed `PutAccount` and `GetAccount` and
nothing else, so there was no state in which spore.host could conclude "this
account is gone" — and everything left behind was therefore permanent.

Most of that residue is harmless. One piece is not: a Route53 A-record under
`{base36}.spore.host` outlives the instance it named, and the public IP it points
at returns to the general EC2 pool. Once AWS reassigns that address, the record
resolves to a stranger's instance. That inverts the usual instinct — the security
consequence argues for expiring DNS records **sooner**, not keeping them around
just in case.

Note what we *cannot* expire: `spawn-ttl-reaper-ec2` is created in the
**customer's** account. We hold only the trust relationship. Only they can delete
it, and an IAM role with no caller is inert and free, so no cost pressure ever
forces the issue.

### The signal is already paid for

The reaper assumes every account's role every 10 minutes. That probe already
distinguishes exactly the states that matter:

- assume **succeeds**, zero managed instances for N days → *dormant but reachable*
- assume **fails** while other accounts succeed → *the customer uninstalled*

Role deletion is the natural uninstall gesture. No new API, and no customer action
beyond what they would already do.

### The four states

Each state exists to answer one question: **is it safe to delete this account's
DNS records?**

| Status | Meaning | DNS expiry eligible |
|---|---|---|
| `active` | assume-role works (also: any legacy row with no `status`) | no — in use |
| `unreachable` | assume-role failed K consecutive runs *while others succeeded* | **no** — see below |
| `dormant` | assume-role **works**, zero managed instances for N | yes — emptiness proven |
| `offboarded` | a human said so | yes — intent stated |

`DNSExpiryEligible()` is the single gate. Unknown/future status values return
false, so a typo can never authorize a deletion.

### Two traps this design refuses to fall into

**1. Correlated failure is indistinguishable from mass uninstall.** The reaper's
role ARN embeds a CloudFormation-generated physical ID
(`…TTLReaperFunctionRole-ZJ84YZ2dCPei`). Recreating that stack changes the suffix
and breaks **every** customer's trust policy simultaneously. So `ApplyProbes`
**changes nothing when every probe fails** — an observation explainable by our own
breakage is not evidence about the customer. This is the same instinct as the DNS
sweep's existing refusal to delete against a partial live set.

The sharp edge is the single-account deployment, where "all probes failed" and
"this one account failed" are the same observation. The guard resolves it as
uninformative, by design (`TestApplyProbes_SingleAccountFailureIsUninformative`).

**2. We lose the ability to verify at the exact moment we would act.** "Only the
role is left, nothing else" needs a *working* assume-role to confirm —
`DescribeInstances` is how emptiness is established. Once the role is gone that
check is impossible. Hence `unreachable` deletes nothing: it is simultaneously the
state we would most like to clean up and the state where we can no longer prove
anything. Those records surface instead through the reaper's report-only
[unmanaged-subdomain signal](https://github.com/spore-host/spawn/blob/main/lambda/ttl-reaper/README.md)
for a human to resolve.

### Policy

`DefaultLifecyclePolicy()` is **K=6** consecutive failed runs (one hour at the
reaper's `rate(10 minutes)`, long enough to ride out a transient STS blip) and
**N=30 days** reachable-and-empty (long enough that a researcher between jobs is
not deprovisioned mid-project). Both are fields on `LifecyclePolicy`.

Other properties worth knowing:

- **`unreachable` stops counting into the reaper's `Errors`.** That field means
  "investigate this" and is worthless once it holds a permanent expected failure —
  a forever-failing assume-role trains the operator to ignore the count.
- **A never-used account is not instantly dormant.** Absent evidence is not
  evidence of absence, so the N-day clock is seeded from first observation.
- **Recovery is automatic, revival is not.** A `dormant` or `unreachable` account
  that runs a spore again returns to `active`. An `offboarded` one never does —
  that status was a human decision and only a human (a re-onboard) undoes it.
- **An unparseable timestamp never triggers a transition.** Acting on a value we
  cannot read is how a healthy account gets deprovisioned by a formatting bug.

### Schema compatibility

Every lifecycle field is `omitempty`, so rows written before this change unmarshal
to the zero value. `AccountStatus()` maps `""` → `active`. **No backfill migration.**
Read the status through that method, never off `.Status` directly, or legacy rows
look like an unknown state.

### Persistence

- `ListAccounts` — paginated Scan. One row per onboarded account (tens, not
  millions). Paginated anyway because a truncated Scan would look exactly like
  "these accounts no longer exist", the precise false conclusion this work exists
  to prevent.
- `UpdateLifecycle` — `UpdateItem`, writing **only** the lifecycle attributes.
  Not a Put, in two directions: a Put would clobber a concurrent re-onboard's fresh
  ExternalId/roleArn with the stale copy a reaper run happened to read, and because
  UpdateItem upserts, it would also resurrect a row deliberately deleted between
  the Scan and the write. Empty timestamps are omitted rather than written as `""`,
  since the dormancy math reads them back.
- `Offboard` — the explicit human transition.
- **No `DeleteItem`.** The registry row costs nothing and is the audit trail;
  offboarding is a status transition, not a deletion.

### Not yet wired

`ApplyProbes` has no caller in production yet — the reaper lives in the `spawn`
repo and would need to collect `ProbeResult`s per run, plus `dynamodb:Scan` and
`dynamodb:UpdateItem` on this table in **its** execution role. Those IAM grants
land with that wiring rather than here, so the policy never carries permissions
with no caller. Tracked in spawn#457.

## Tests

```bash
go test ./...
```

Pure logic (`validate`, `ApplyProbes`, `DNSExpiryEligible`) needs no AWS. The
persistence tests run against substrate's in-memory DynamoDB
(`substrate.StartTestServer`), not a real table.

The state machine's guards are mutation-verified: neutering the correlated-failure
guard, the K threshold, the N threshold, or either `offboarded` exclusion each
fails a test. One redundancy is documented in place — the never-used-account clock
seed reaches the same result as the unparseable-timestamp branch, and is kept
because "never observed" and "observed but unreadable" are different facts that
happen to share a remedy.

## Build & deploy

```bash
make build            # linux/arm64 bootstrap + function.zip
make upload           # → s3://spawn-binaries-us-east-1/portal-phone-home/
```

Infrastructure is OpenTofu, not the Makefile: `infra/tofu/portal-phone-home/`
(Function URL, execution role, the DynamoDB table with PITR + a customer-managed
CMK, and a second CMK for env encryption).
