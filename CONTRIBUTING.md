# Contributing to spore.host

Thanks for your interest in improving spore.host. This repo (`spore-host`) is the
umbrella: it holds the documentation site, the hosted API/dashboard, deployment
automation, and this project-wide guidance. The individual tools live in their own
repositories.

## Where things live (open issues in the right repo)

| You want to… | Repo |
|--------------|------|
| Report a docs problem, or a website/dashboard/API issue | [spore-host](https://github.com/spore-host/spore-host) (this repo) |
| File a bug or request a feature in **spawn** (launch/lifecycle) | [spawn](https://github.com/spore-host/spawn) |
| …in **truffle** (discovery/pricing/quotas) | [truffle](https://github.com/spore-host/truffle) |
| …in **lagotto** (capacity watching) | [lagotto](https://github.com/spore-host/lagotto) |
| …in the **MCP server** | [spore-host-mcp](https://github.com/spore-host/spore-host-mcp) |
| Add or fix an official **plugin** | [spore-plugins](https://github.com/spore-host/spore-plugins) |
| A **workflow adapter** (Nextflow/Snakemake/CWL/WDL/Airflow) | its own `*-spawn` / `*-executor-plugin-spawn` repo |

Not sure which repo? Open it here and we'll route it, or ask in
[Discord](https://discord.gg/2deGRFCW).

## Getting help vs. reporting

- **Question or usage help:** [Discord](https://discord.gg/2deGRFCW) is faster than an issue.
- **Bug / feature:** open an issue in the relevant repo (templates provided).
- **Security vulnerability:** do **not** open a public issue — use the private
  [security advisory form](https://github.com/spore-host/spore-host/security/advisories/new).
  See [SECURITY.md](SECURITY.md).

## Development

Each Go tool follows the same conventions (see the repo's `CLAUDE.md` / `Makefile`):

- **Build/test:** `make check` (fmt, vet, lint, short tests) before every commit;
  `make test` for full coverage; `make build` to build.
- **Style:** `gofmt`/`goimports`; pass `go vet` + `staticcheck` + `golangci-lint`;
  godoc comments on exported identifiers; wrap errors with `fmt.Errorf("op: %w", err)`.
- **Commits:** [Conventional Commits](https://www.conventionalcommits.org/)
  (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`), branch prefixes `feat/` `fix/`
  `docs/`, one PR per change, link the issue.
- **Branching:** always cut from `origin/main` with a clean tree, and run
  `scripts/branch-preflight.sh` before opening the PR — see
  [Branching and merging](#branching-and-merging) below.
- **Versioning:** [SemVer 2.0.0](https://semver.org) + a Keep-a-Changelog
  `CHANGELOG.md`; update `## [Unreleased]` in the **same PR** as any user-facing change.
- **Docs:** the site is VitePress under `docs/`. `cd docs && npm install && npm run
  docs:dev` to preview; `npm run build` must pass. CLI references are generated —
  don't hand-edit generated pages.

## Branching and merging

`main` is **squash-merged and linear** — one commit per PR, no merge commits. It
also **auto-deploys**: `web/**` publishes to spore.host and `docs/**` to
docs.spore.host on push. So a merge is a release.

Start every branch the same way:

```bash
git fetch origin main
git status --porcelain                        # must be empty
git checkout -b fix/123-short-name origin/main
scripts/branch-preflight.sh                   # before you open the PR
```

Being a few commits behind `main` is fine and needs no rebase; squash-merge handles
it. What is **not** fine is branching from a dirty tree or from a `HEAD` that holds
unmerged work — the new branch silently adopts those commits, and whichever PR
merges first ships them under its own number. The second one then merges **empty**.

### What `main` enforces

`main` is protected: no force pushes, no deletion, linear history required, and
these checks must pass before a merge —

| Required check | Workflow |
|---|---|
| `Branch hygiene` | `ci.yml` |
| `Docs build + link check` | `ci.yml` |
| `Go Vulnerability Check` | `security.yml` |
| `Secret Scan (gitleaks)` | `security.yml` |
| `Trivy Security Scan` | `security.yml` |
| `Semgrep SAST` | `security.yml` |

No review is required — the repo is effectively single-committer, so requiring one
would only mean overriding it every time.

Two deliberate omissions, both of which would otherwise deadlock every PR:

- **`Lambda module tests` is not required.** It's a matrix job, so GitHub appends the
  matrix values to the check name (`Lambda module tests (lambda/rest-api, 10)`).
  That `10` is the coverage floor — editing any floor renames the check, and a
  required check that never reports blocks all PRs until protection is edited to
  match. It still runs and still reports on every PR.
- **`web-ci.yml` and `ci-runner-drift.yml` are not required.** Both are
  `paths:`-filtered, so they don't run on a PR that misses their directories. A
  required check that never fires is indistinguishable from one still pending.

`enforce_admins` is **off**, so an admin can still push directly or merge red. That
is a deliberate escape hatch for an urgent fix, not an invitation — the checks exist
because `main` deploys.

That happened: #521 was cut from a tree still holding #519's commit, merged first,
and carried it. #519 landed as an empty commit — which matched neither the
`web/**` path filter on `deploy-site.yaml` nor `web-ci.yml`, so the PR that existed
to fix two user-facing strings ran no web tests and triggered no deploy. Nothing was
red; the work simply wasn't where its PR said it was.

After merging:

- Check the merge commit isn't empty when it should have shipped something:
  `git show --stat <sha>`.
- For `web/**` or `docs/**`, confirm the deploy workflow ran —
  `gh run list --commit <sha>`. A path-filtered workflow that never fires looks
  exactly like one that passed.
- **Delete your branch.** Auto-delete-on-merge is off, so they accumulate.

To tell whether an old branch is safe to delete, compare its tip to the head SHA
GitHub recorded for its merged PR (`gh pr list --state merged --json
headRefName,headRefOid`). Don't trust `git branch --merged` — squash-merge breaks
ancestry — and don't read `git diff main <branch>`, where a stale branch appears to
delete every file `main` has gained since.

## Cost safety (important)

Several tools launch **real, billable** AWS resources. Any test that touches AWS
must set a TTL, terminate explicitly when done, and be independently leak-checked
(no orphaned instances). Never commit changes that could leave resources running.

## Code of Conduct

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

Contributions are accepted under the project's [Apache 2.0 license](LICENSE).
