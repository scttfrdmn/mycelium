# Changelog

Notable changes to the **spore.host shared infrastructure** repo — the hosted
REST API, dashboard, Python SDK, deployment automation, AMI builds, the
`spore-bot` Slack/Teams Lambda, and the documentation site.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
Unlike the CLI tools (truffle/spawn/lagotto), this repo is **not tag-released** —
its Lambdas and site deploy continuously — so changes are grouped under
`Unreleased` until a milestone warrants a dated section. See the per-tool repos'
own changelogs for CLI releases.

## [Unreleased]

### Added
- spore-bot formats the new `pre_stop_failed` / `pre_stop_timeout` lifecycle
  events (spawn#186): a failed or timed-out `--pre-stop` hook now shows as a
  loud orange/red Slack/Teams/Discord message (and SMS) carrying the hook's
  error/output tail, instead of being indistinguishable from a clean shutdown.
  Surfaces the spawn#184 data-loss shape (a pre-stop that "succeeded" saving
  nothing).

### Removed
- Dead `Registry.RedeemConnectCode` in spore-bot (audit L-health, #374). Connect
  codes are redeemed on the spawn side (`spawn bot register --connect-code`,
  which atomically deletes the shared-table item); the Lambda only issues them.
  The duplicate, never-called Lambda-side redeem method is removed to avoid
  misrepresenting the flow.

### Fixed
- **`extend` can no longer prematurely reap an instance** (audit M-corr, #374).
  The bot `/spore extend`, the REST API `extend` action, and the SMS `extend`
  reply now floor the new TTL deadline at `now + requested-duration`. Previously,
  if the instance had a missing/unparseable `spawn:ttl` (deadline anchored to a
  long-past launch time) or an already-expired `spawn:ttl-deadline`, the
  recomputed deadline could land in the past — terminating the instance at the
  moment the user asked to keep it alive. An extend now always grants at least
  the requested duration from the current moment.

### Security
- **Medium/Low audit hardening** (#374): the OAuth state HMAC now fails closed
  when `BOT_OAUTH_SECRET` is unset or still `change-me` (the old default let a
  forged state complete the flow); Twilio webhook signature verification is
  **required in production** (`SPORE_ENV=production`) and the
  `SKIP_TWILIO_SIGNATURE` escape hatch is ignored there; connect codes are now
  generated with `crypto/rand` (8 hex chars, ~4.3B) instead of a time-seeded
  value; and `MarkTerminated`'s DynamoDB retention now matches its documented
  7 days (was 24h).
- **Hosted REST API now enforces per-project tenant isolation** (audit C1, #369).
  Previously every handler received a validated API-key principal but never used
  it, so any valid key could list/launch/stop/terminate/extend **every** instance
  in the account. Launches are now stamped with `spawn:project=<key's project>`,
  and list/get/stop/start/hibernate/terminate/extend are scoped to the
  principal's project (fail-closed: a key with no project can't launch or reach
  any instance; non-owned instances return 404, not 403, so existence isn't
  leaked). **Operator note:** instances launched before this change carry no
  `spawn:project` tag and become invisible to the API — backfill the tag
  (`aws ec2 create-tags --tags Key=spawn:project,Value=<project>`) to re-expose
  them.
- **SMS "extend" reply now writes `spawn:ttl-deadline`** (not just `spawn:ttl`,
  which spored ignores — a silent no-op, same class as #371) and is capped at the
  7-day maximum.
- **Teams Bot Framework requests now fully validate the bearer JWT** (audit H4,
  #372). Previously a `Bearer …` request was trusted as long as the server-side
  `TEAMS_APP_ID` env was set — no token validation at all — so any caller of the
  public Function URL could forge a Teams activity. The token is now verified for
  RS256 signature against Microsoft's published JWKS, issuer
  (`https://api.botframework.com`), audience (`== TEAMS_APP_ID`), and expiry;
  `alg:none`/HMAC-confusion are rejected. Verification fails closed.
- **Slack/Teams signature verification now rejects an empty signing secret**
  (audit H5, #373). HMAC with an empty key is forgeable, and OAuth-installed
  Slack workspaces persist no per-workspace secret. Slack now falls back to the
  app-level `SLACK_SIGNING_SECRET` env (the secret is app-level, not
  per-workspace), and both verifiers fail closed when no secret is available.
- **Hosted REST API now enforces lifecycle bounds** (audit H3, #371). Unlike the
  CLI, the API called the spawn client directly and bypassed the 1h-idle
  zombie-prevention default, so an empty-TTL launch produced an instance with no
  deadline and no reaper tag. Launches with neither TTL nor idle timeout now get
  a default idle timeout, all TTL/idle/extend durations are capped at a 7-day
  maximum, and the `extend` action now writes `spawn:ttl-deadline` (not just
  `spawn:ttl`) so the extension actually takes effect (it was a silent no-op,
  same class as spore-host-mcp#11). The `/spore extend` and `/spore idle` bot
  commands gained the same deadline fix and 7-day cap.
- **spore-bot `/notify` now gates per-user DM and SMS fan-out on instance
  registration** (audit C2, spore-host/spawn#369-370 class). Previously the
  endpoint only checked that `workspace_id`/`instance_id` were non-empty, so
  anyone who learned an instance_id + workspace_id could trigger DMs to
  registered users and platform-billed SMS for an instance that wasn't theirs.
  DM/SMS now require the instance to be registered in the workspace
  (`InstanceRegisteredInWorkspace`); the channel-webhook path is left open (the
  workspace owner opted in, no per-user targeting or SMS cost). PKCS#7 identity
  verification is wired in as log-only for now (the embedded-cert path is
  unreliable cross-region; #294) and will flip to hard-reject once certs are fixed.

### Security
- Security CI hardened to a consistent gate across the suite: govulncheck now
  scans **all** Go modules (added `spore-bot` — previously only `rest-api`),
  added **gitleaks** secret scanning (MIT binary; org-license-free; allowlist for
  doc examples + test fixtures), and Trivy's filesystem scan now includes the
  **secret** scanner. The same Security workflow (govulncheck/gitleaks/Trivy/
  Semgrep) was added to the previously-unscanned tool repos (spawn, truffle,
  lagotto, nf-spawn, spore-host-mcp).

### Added
- **Infrastructure as code (OpenTofu), starting with spore-bot.** New
  `infra/tofu/spore-bot/` module — the first IaC in the umbrella — reconciles the
  previously hand-deployed spore-bot Lambda + Function URL under OpenTofu via
  `tofu import` (imported to a zero-functional-diff plan; only additive
  `managedby=opentofu` tags). Code and secret env vars stay out-of-band
  (`ignore_changes`), so deploys and secrets are untouched. Reference pattern for
  migrating the rest of the hand-rolled `setup-*.sh` infra.

### Fixed
- **spore-bot** Discord slash-command results now appear reliably: the async
  executor could PATCH the interaction's response before Discord registered the
  deferred ack (a 404 race — `/spore help` showed "thinking…" then nothing). The
  follow-up now retries a 404 with short backoff (#2).
- **spore-bot ran under prism-bot's IAM role** (`prism-bot-PrismBotFunctionRole`),
  a cross-project coupling that, among other things, denied writes to the
  `spore-bot-audit` table. Created a dedicated least-privilege **`spore-bot-role`**
  and repointed the function; spore.host's bot no longer borrows prism's identity.
- **spore-bot** delivers Discord lifecycle notifications (Phase 1 of
  spore-host/spawn#2): when an instance's notify platform is `discord`, the
  `/notify` handler posts a color-coded Discord embed (severity-colored, with
  instance/region/URL fields) to the workspace's channel webhook. Adds a
  `PublicKey` field to the workspace registry for Discord's Ed25519 interaction
  verification (used by Phase 2 slash commands). New `docs/guides/discord-setup.md`.
- **spore-bot** Discord slash commands (Phase 2 of spore-host/spawn#2): a
  `/discord` interactions endpoint verifies Discord's Ed25519 request signature
  (per-application public key), answers the PING/PONG handshake, and dispatches
  `/spore list|status|start|stop|hibernate|url|extend|connect` through the same
  async action machinery as Slack/Teams — replying with a deferred ack and
  editing in the result (meeting Discord's 3-second deadline). Multi-tenant: any
  guild installs the published app and registers via `spawn notify workspace-add
  --platform discord`. New `scripts/register-discord-commands.sh` registers the
  global slash command; setup guide extended with the Phase 2 install flow.
- **spore-bot** honors the friendly account-name DNS segment: it displays
  `{name}.{account-name}.spore.host` when the instance has a `spawn:account-name`
  tag (falling back to base36) and matches a user-typed target against either
  form (spore-host/spawn#121, #357 / #358).
- README documents Windows support (ISO → AMI → launch → RDP/SSH) and a Quick
  Start example (#355).
- `CLAUDE.md` records the project-wide **SemVer 2.0.0 + Keep a Changelog** policy
  that applies to every spore.host repo (#355).
- CI runs per-Lambda-module tests and bootstrapped `rest-api` Lambda coverage
  (#337).

### Changed
- Relocated the `spore-bot` Lambda from the spawn repo into this infra monorepo.
- Bumped `codecov/codecov-action` v5 → v7 (#350).

### Removed
- Untracked the 18 MB committed `lambda/spore-bot/spore-bot` build artifact (now
  gitignored; regenerated by the build).

---

Earlier history is in the
[commit log](https://github.com/spore-host/spore-host/commits/main) and the
[pull requests](https://github.com/spore-host/spore-host/pulls?q=is%3Apr+is%3Amerged).
