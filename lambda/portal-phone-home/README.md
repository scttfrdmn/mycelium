# portal-phone-home

The spore.host portal's BYOA onboarding registrar.

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

## What lives elsewhere

The registry schema and the account lifecycle state machine (spawn#457) are in
[`../accountlifecycle`](../accountlifecycle/), not here. This module registers
accounts; [`../portal-account-prober`](../portal-account-prober/) probes them and
writes the lifecycle transitions.

That split exists so `ApplyProbes` has exactly one copy — its refusals are
mutation-verified, and a second copy would drift out from under those tests. The
`accountlifecycle` README is where the four statuses, the `DNSExpiryEligible` gate,
and the three false-deprovision traps are documented.

## Tests

```bash
go test ./...
```

`validate()` is pure and needs no AWS. That is now the whole of this module's own
test surface — the registry round-trip tests moved to `accountlifecycle` along with
the code they cover, which is why the CI coverage floor for this module is lower
than it was.

## Build & deploy

```bash
make build            # linux/arm64 bootstrap + function.zip
make upload           # → s3://spawn-binaries-us-east-1/portal-phone-home/
```

Infrastructure is OpenTofu, not the Makefile: `infra/tofu/portal-phone-home/`
(Function URL, execution role, the DynamoDB table with PITR + a customer-managed
CMK, and a second CMK for env encryption).
