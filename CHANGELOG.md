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

### Fixed
- **Docs: MCP scope, SSH-key wording, and reaper guarantee-by-mode** (from a second
  external review). Reworded the How It Works MCP section — it no longer says the
  server "exposes all of the above"; it now states MCP does read/manage operations
  (find, status, stop, terminate, extend) and **cannot launch** by design. Corrected
  the SSH-key note in `how-it-works.md` and `architecture.md`: spawn imports your
  existing default public key and only generates/manages one under `~/.spawn/keys/`
  if you have none (never touches private keys) — matching the FAQ, aws-auth, and
  quickstart pages. Added the **lifecycle guarantee-by-deployment-mode** table
  (CLI-only / self-hosted backstop / hosted integrations, with the reaper's dry-run
  default) to the Safety page, previously only in `SECURITY.md`.
- **Docs: corrected `--cost-limit` description** (safety, glossary, spored, EC2
  tags). It's a **compute-only** ceiling (excludes EBS/storage), accumulates
  across stop/resume rather than resetting, **terminates** the instance, and fires
  independently of the TTL (first ceiling to fire wins). Matches the spored fix in
  spawn PR #400 (enforcement was resetting the cost clock on each restart).

### Added
- **Docs: execution-fabric restructure (plugins & workflow adapters).** Following a
  second review of the extension subsystem, regrouped the docs so spawn reads as an
  execution substrate with four extension layers — **Extend an instance** (instance
  plugins), **Run many jobs** (sweeps, arrays), **Coordinate multiple steps**
  (instance queues, pipelines, MPI), and **Workflow adapters** — instead of a flat
  "Advanced" list. New page **[Which execution tool?](https://docs.spore.host/guides/choosing-execution)**
  with two decision matrices (layer→question, need→tool) plus per-task-EC2 fit
  guidance. Renamed "Plugins"→"Instance plugins" and "Workflow Engines"→"Workflow
  adapters" (CLI flags unchanged).
- **Docs: parameter-sweeps now document existing concurrency/recovery controls** —
  `--max-concurrent`, `--budget`, `spawn sweep resume` (checkpoint), and
  `spawn sweep collect`, which the guide previously omitted.
- **Docs: instance-queue operational detail** — per-job `retry`/`env`/`result_paths`
  fields, per-job log paths, resume-skips-completed behavior, and Spot-interruption
  guidance, all verified against the spored queue runner.

### Changed
- **Docs: corrected workflow-adapter maturity.** The overview no longer calls the
  five engine integrations "first-class"; it now leads with a **status &
  compatibility matrix** (nf-spawn = experimental prototype; miniwdl = early,
  validation in progress; CWL/Snakemake/Airflow = v0.1.0 initial releases) and
  distinguishes "native adapter" from "production-ready." Experimental badges added
  to the Nextflow guide.
- **Docs: job-array `--min-viable` semantics clarified** — `{total}`/`JOB_ARRAY_SIZE`
  is the *requested* count and partial launches leave *sparse* indexes; shard
  schemes must not assume a dense range.

### Fixed
- **Docs: pipeline manual example was misleading** — clarified that `--on-complete`
  does not launch the next stage and that running `spawn launch` commands by hand
  is not a pipeline (the Lambda orchestrator chains stages from the DAG definition).
  Scoped `spawn pipeline` to coarse DAGs (not a scientific workflow engine) and
  marked stream mode (tcp/grpc/zmq) Experimental with its operational caveats.
- **Docs: instance-queue (batch-queue) sequencing contradiction** — stated plainly
  that jobs run strictly one at a time in topological order and `depends_on` is
  ordering only, not concurrency.
- **Docs: added a plugin Trust & permissions section** — installing a plugin runs
  its author's code locally and as root on the instance; documents what `github:`
  refs resolve to, the minimal-env/`env_passthrough` limit, `spawn plugin validate`,
  and links the tracked inspect/permissions gaps.

### Added
- **Docs: researcher-facing information-architecture overhaul.** Restructured the
  docs site around user intent instead of product structure, following an external
  review. New top-level nav: Introduction → Start Here → Common Workflows → Tools →
  Automation → Chat & AI Control → Administration → Reference (one global sidebar).
  New pages: **Security, credentials & data flow** (`/architecture`, "what runs
  where" trust/data-flow model), **Costs & safety guarantees** (`/safety`,
  auto-terminate failure boundaries + pre-launch cost estimation), **Waiting for
  scarce capacity** (a Lagotto first-use tutorial), **Glossary**,
  **Troubleshooting & common mistakes**, and **Event schemas**. The Guides landing
  is now three narrative user stories. Each tool page opens with a consistent
  *What it is / When to use / First commands* trio; Truffle gains a find-vs-search
  decision box, Spawn a three-tier progressive-disclosure map and a spawn-vs-spored
  "what runs where" diagram. Added audience/maturity badges (Beginner / Advanced /
  HPC / Automation / Stable / Beta …) with WCAG-AA colours in light and dark mode.
- **Docs: heterogeneous parameter sweeps** (spawn#372) — the sweeps guide now
  shows varying `instance_type`/`ami`/`spot` per entry (price-performance
  benchmark example), with the per-entry AMI auto-detection, the one-OS-per-sweep
  rule, and the detached-sweep `ami:` caveat.
- **Zenodo DOI**: spore.host is now archived on Zenodo with a citable DOI
  (concept DOI [10.5281/zenodo.21439339](https://doi.org/10.5281/zenodo.21439339),
  always latest). Wired into `CITATION.cff`, `codemeta.json`, and a README badge +
  Citation section.
- **`codemeta.json`** — CodeMeta (JSON-LD / schema.org) software metadata, for
  software registries and citation tooling; complements `CITATION.cff`.
- **`CITATION.cff`** — machine-readable citation metadata (GitHub renders a
  "Cite this repository" button). Foundation for the Zenodo DOI integration.
- **Docs AI-readiness: `llms.txt` manifest** (#424) — a curated
  [llmstxt.org](https://llmstxt.org) entry map, generated at build time from the
  sidebar (a `buildEnd` hook) so it can't drift from the nav.
- **Docs discoverability: `sitemap.xml`, per-page descriptions, OpenGraph** (#425)
  — set the VitePress `sitemap.hostname` (`docs.spore.host`); added a
  `description:` to ~40 content pages (→ real `<meta name="description">` and
  better AI/search snippets); added default OG/Twitter card `head` tags.

### Fixed
- **Site: resolved the "five vs six tools" inconsistency.** The marketing site
  now says "Six tools" and includes a **Spored** card (it was omitted); the docs
  already said six. `web/README.md`'s stale tool list updated to match.
- **Site: consistent HTTP API status.** The API is now labelled **beta / used by
  the SDK** everywhere — removed "coming soon"/"on the roadmap" framing on
  `web/library.html` (added an inline **Beta** badge to the endpoint block so it
  doesn't read as a live public contract) and reconciled the `guides/python-sdk`
  and `guides/self-hosting` wording.
- **Copy: clarified spore.host is not itself the host** — the hero, docs home,
  and How It Works now state up front that it runs on your own AWS account with
  your own credentials, with no new provider to sign up for.
- **Docs accessibility: WCAG AA contrast** (#423) — tool badges are small
  (0.7rem) bold text, so they need 4.5:1: several failed in light and/or dark
  mode. Badge text now uses per-mode colors that all clear 4.5:1, and dark-mode
  link text uses a lightened brand blue (the `#4059E5` override dropped to 3.09:1
  on the dark background); the hero brand button keeps its solid blue + white.

- **Docs: job arrays now document `--min-viable`** (spot partial-success floor)
  and parameter sweeps document `spawn alerts` (completion/failure/cost-threshold
  notifications via email/Slack/SNS/webhook) — both features the reference listed
  but no guide explained.

### Fixed
- **Docs: corrected the SSH/connectivity FAQ** — `spawn connect` logs you in as
  your own username (the instance matches your local user) and reuses your
  `~/.ssh/id_ed25519`/`id_rsa` key when present; added SSM/private-subnet and
  common-failure troubleshooting. Replaces the stale `ec2-user`/`spawn-default`
  and `--key-name` guidance.
- **Docs: corrected the Pipelines guide** — `spawn pipeline launch` takes a JSON
  definition file (not the fictional YAML + `--slack-workspace`/`--efs-id`
  flags), stages form a DAG via `depends_on`, and data is handed off with
  `data_input`/`data_output`. Also fixed `status`/`cancel`/`collect` (take a
  pipeline id) vs `graph` (takes the file).
- **Docs: rewrote the lagotto SageMaker section** — `--service sagemaker` now
  **submits your training job** (`--sagemaker-config`) when the EC2-family proxy
  fires; it is no longer notify-only. `--action notify` (alert only) and `spawn`
  (submit) are both valid; `hold` is rejected.
- **Docs: fixed the sidebar** — wired the orphaned Discord setup guide into nav
  and de-duplicated the two "Self-Hosting" entries into one group.

### Added
- **Docs: new "Managing Instances & Data" guide** covering the operational spawn
  commands that had no prose — `stage` (cross-region S3 staging), `snapshot` +
  `launch --attach-volume` (large reference data as an EBS volume),
  `upgrade-spored`, and `resources`/`orphans` (inventory + cost hygiene).
- **Docs: complete `plugin.yaml` field reference** in the Plugins guide (top-level,
  config params, conditions, local/remote phases, step fields, outputs) derived
  from the loader schema.
- **Docs: truffle overview now documents the shared `--profile`/`--account`
  config** and links the AWS auth guide.
- **Docs: the CLI command/flag reference is now generated + drift-gated.** Each
  CLI (spawn/truffle/lagotto) emits its exhaustive reference from the binary
  (`libs/docgen`); the umbrella vendors those fragments into `docs/gen/<cli>/`,
  and `docs/tools/reference/<cli>.md` collapsed to a thin prose shell that
  `@include`s them (spawn's reference dropped from ~1056 hand-maintained lines to
  a short shell). A new `sync-cli-docs.yaml` workflow re-vendors each CLI's
  reference from its latest release tag (fired by the CLI release, on demand, or
  weekly) and opens a PR, so the site's reference tracks the shipped binaries
  automatically. A `docs-build` CI job now gates the docs (VitePress build +
  `@include`/link check) — the site previously had no PR gate. (2026-07 docs audit.)

### Changed
- **Docs: code/command text now renders in Atkinson Hyperlegible Mono**, matching
  the body's Atkinson Hyperlegible for a consistent, high-legibility monospace.

### Security
- **spore-bot: removed the static `BOT_EXTERNAL_ID` cross-account fallback (fail
  closed).** Assuming a registration's cross-account role now requires that
  registration's own per-account `external_id` (generated at register time since
  the per-registration ExternalId change); a registration without one can no
  longer assume its role via the shared `spawn-bot` value, which has been
  removed. Safe now that both bot deployments run the per-account read path and
  no live registration depends on the shared value. Any account's
  `SpawnBotCrossAccount` trust policy must require its per-account ExternalId
  (spawn CFN `ExternalId` parameter) before that account's instances can be
  controlled (#413, #374).
- **rest-api: API keys are now stored and looked up by SHA-256 hash.** The raw
  `sk_...` key was the DynamoDB partition key, so a table dump yielded usable
  credentials. `validateAPIKey` now hashes the presented key and looks it up by
  hash; legacy plaintext-keyed rows still authenticate (dual-read) and are
  rewritten to their hashed form on first use, so the migration is transparent
  and needs no client changes. New keys must be inserted hashed (see
  `infra/DEPLOY.md`) (#374).
- **rest-api: request log now includes a truncated, non-secret KeyID.** Each
  request logs `keyid=` (first 8 hex chars of the key's SHA-256) for
  attribution; the previous `Principal.KeyID` exposed the raw key's first 8
  chars, which leaked the secret's prefix (#374).
- **spore-bot: cross-account role assumption now uses a per-registration STS
  ExternalId.** Registrations get a high-entropy `external_id` (generated at
  register time, or supplied by the admin so it can be pre-baked into the
  customer role's trust policy), and the assume uses it. (The shared
  `BOT_EXTERNAL_ID` fallback that initially backed this has since been removed —
  see above; EC2 `Resource:"*"` scoping is opt-in via the spawn CFN template.)
  (#374).
- **spore-bot: removed the dead log-only instance-identity verification.** The
  PKCS#7/embedded-cert check in the notify path (`verifyNotifyAuth` + embedded EC2
  certs, `signature.go`) never rejected anything — it only logged — and its
  embedded certs were unreliable across regions/rotations (same reason the
  dns-updater cert path was retired, #294). Keeping it misrepresented the crypto
  posture. The enforced control is (and was) the registry-membership gate: a
  notification is only delivered for an instance registered in the target
  workspace. Removed the file, the log-only call, the now-unused `NotifyRequest`
  identity fields, and the `fullsailor/pkcs7` dependency (#374).

### Fixed
- **spore-bot: `GetWorkspacesForPlatform` no longer hides a failed DynamoDB scan.**
  It previously returned "no workspaces" identically for an empty result and a
  scan error, masking outages; the error is now logged (#374).

### Changed
- **rest-api SMS handler owns its pending-reply types locally.** spawn is
  removing its dead `pkg/sms` (spawn#293); rest-api was the only cross-repo
  consumer and used only the inbound types (`PendingKey`/`PendingNotification`/
  `PendingTable`). Those now live in `lambda/rest-api/sms.go` (mirroring
  `lambda/spore-bot/sms_notify.go`, which already keeps its own copy); the
  `pkg/sms` import is dropped. No behavior change; rest-api still depends on
  spawn's `pkg/aws`/`pkg/config`.

### Security
- **Pinned the CI Go toolchain to 1.26.5** to clear GO-2026-5856, a `crypto/tls`
  standard-library advisory present in go1.26.4 (affects the Go Lambdas / modules
  built by CI). govulncheck is green again.

### Added
- **`infra/ci-runners/` — the self-hosted CI runner fleet is now versioned** (it
  previously lived only on orion.local, so fixes weren't reviewable) (#381). The
  Dockerfile, self-healing `entrypoint.sh`, compose, boot launchd unit, and a
  `boot-recreate.sh` recovery/boot script, with a README documenting the failure
  modes and operations.

### Fixed
- **CI runner fleet no longer fills its disk and orphans jobs** (#381). The
  ephemeral runners ran under `restart: always`, which restarts each finished
  container **in place** — so its writable layer (`_work` + a ~3.5GB Go build
  cache) grew unbounded across cycles (~7GB × 6) until the colima volume hit 80%+
  and a job mid-run ran out of space, dying with no recorded steps (a ~10-min
  GitHub timeout — this is what spuriously failed spawn#258). The entrypoint now
  self-heals each cycle (clears `_work`, caps the go-build cache, `--replace`
  re-registers), and a launchd boot unit recreates the fleet (not restart-in-place)
  after a host reboot, fixing the prior reboot crash-loop too. Recovery on the
  host reclaimed ~38GB (80%→13%).

### Fixed
- Corrected broken GitHub and pkg.go.dev links on the website and docs that
  assumed a monorepo layout: the tools are split repos, so `truffle`, `spawn`,
  `lagotto`, and the MCP server now link to their own `github.com/spore-host/*`
  repos (and Go imports use `github.com/spore-host/<tool>/...`, not a nested
  `spore-host/spore-host/...` path).
- Python SDK docs/site no longer say "coming soon" / "not on PyPI yet" — the
  `spore-host` package is published (`pip install spore-host`); point SDK links
  at the standalone `spore-host/python-sdk` repo.
- Fixed the `scttfrdmn/tap` Homebrew command on the dashboard (→ `spore-host/tap`)
  and added the missing `docs/public/favicon.svg`.

### Added
- **`infra/tofu/dns-updater`** — OpenTofu module bringing the hand-deployed
  `spawn-dns-updater` Lambda (Route53 record updater) under IaC, mirroring the
  `spore-bot` import-onto-live pattern (imports to a near-zero diff: only additive
  `managedby` tags). This is step 0 of the spawn#173 cutover that moves the DNS
  updater off the spoofable instance-identity-document auth onto the Function URL's
  `AuthType: AWS_IAM`. The module carries a gated `enable_iam_invoke` toggle (the
  `Principal: "*"` AWS_IAM invoke grant — scalable, no per-account enumeration) and
  documents the full cutover ordering; the destructive `AuthType` flip stays gated
  on a SigV4-signing fleet being fielded. See the module README.
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
