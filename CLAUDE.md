# CLAUDE.md

## Response Style
- Concise by default. No explanations unless asked.
- No file creation unless explicitly requested.
- Fix bugs silently unless cause is non-obvious.

## Development Workflow

### Planning Mode
**Use plan mode for non-trivial features** - Enter plan mode when:
- Feature spans multiple components or files
- Multiple valid implementation approaches exist
- Architectural decisions need to be made
- User requirements need clarification

**Plan mode process:**
1. Use `EnterPlanMode` tool to explore codebase
2. Launch Explore agents to understand existing patterns
3. Launch Plan agent to design implementation
4. Ask clarifying questions via `AskUserQuestion`
5. Write detailed plan to plan file
6. Use `ExitPlanMode` to get approval before implementing

### Issue Documentation
**Document plans as GitHub issues:**
- Create issue for each major feature/fix before starting
- Include clear scope, acceptance criteria, and tasks
- Link related issues and PRs
- Update issue with implementation notes as you work
- Close with summary comment when complete

**Create milestones for releases:**
- Group related issues under version milestones (e.g., v0.1.0)
- Track progress toward release goals
- Use milestone due dates to guide priorities

**Use labels consistently:**
- `priority:critical`, `priority:high`, `priority:medium`, `priority:low`
- `type:feature`, `type:bug`, `type:refactor`, `type:docs`, `type:test`
- `component:truffle`, `component:spawn`, `component:spawnd`
- Create custom labels as needed for project-specific categories

## Versioning & Changelog
- Follow **[Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html)**: MAJOR for breaking changes, MINOR for backward-compatible features, PATCH for fixes (pre-1.0, breaking changes bump MINOR).
- Maintain a **[Keep a Changelog](https://keepachangelog.com/en/1.1.0/)**-format `CHANGELOG.md` at the repo root.
- Update `CHANGELOG.md` in the **same PR** as any user-facing change: add an entry under `## [Unreleased]` in the right group (Added/Changed/Deprecated/Removed/Fixed/Security). Write for humans — describe the effect, not the implementation.
- On release: promote `## [Unreleased]` to `## [X.Y.Z] - YYYY-MM-DD`, open a fresh empty Unreleased, update the comparison links, and tag `vX.Y.Z`.
- GoReleaser auto-generates the GitHub Release notes from commits; `CHANGELOG.md` is the curated, human-facing source of truth. Keep both.
- This applies to every spore.host repo (truffle, spawn, lagotto, …), each with its own `CHANGELOG.md`.

## Go Standards
- Go 1.21+ with modules
- `gofmt`, `goimports` on all code
- Pass `go vet`, `staticcheck`, `golangci-lint` before done
- Godoc comments on all exported identifiers
- No `panic` except unrecoverable init failures

## Code Style
- Idiomatic short names: `r` for reader, `ctx` for context, `err` for error
- Wrap errors with `fmt.Errorf("operation: %w", err)`
- Return early on errors; avoid deep nesting
- Prefer standard library over dependencies
- Group imports: stdlib, external, internal

## CLI Patterns
- Use `cobra` for CLI structure
- Flags over args when >1 input
- Exit codes: 0=success, 1=error, 2=usage error
- Stderr for errors/logs, stdout for output
- Support `--json` output where applicable

## AWS SDK
- Use `aws-sdk-go-v2`
- Load config with `config.LoadDefaultConfig(ctx)`
- Always pass context for cancellation
- Wrap SDK errors with operation context
- Use pagination helpers for list operations

## Testing
- Minimum 60% coverage, target 80%+
- Table-driven tests as default
- Use `t.Helper()` in test helpers
- Mock AWS with interfaces, not SDK mocks
- Test error paths, not just happy path
- Use `testdata/` for fixtures
- Golden files for complex output verification

## Security
- Never log credentials or tokens
- Use `golang.org/x/crypto` for cryptographic operations
- Validate all external inputs
- Sanitize before logging user-provided data

## Project Structure
- `truffle/` - Instance discovery and quota management
- `spawn/` - EC2 launching and wizard
- Each tool: `cmd/` (commands), `pkg/` (packages)

## Git & GitHub
- Use `gh` CLI for all GitHub operations
- Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`
- Branch naming: `feat/`, `fix/`, `refactor/` prefixes
- PR per feature/fix; link to issue

### Branching: always cut from `origin/main`, with a clean tree

```bash
git fetch origin main
git status --porcelain              # must be empty
git checkout -b fix/123-thing origin/main
```

**Never** `git checkout -b` from a dirty tree or from a `HEAD` carrying unmerged
work. `main` is squash-merged and linear, so a topic branch being a few commits
behind the tip is normal and needs no rebase — but a branch cut from *another
branch* silently adopts that branch's commits.

Run `scripts/branch-preflight.sh` before opening a PR. It checks the four things
that actually broke: not on `main`, clean tree, based on `origin/main` history,
and every commit unique to this branch. `--fix` prints recovery commands.

**Why this is a rule and not a preference.** PR #521 (a CI-runner fix) was branched
from a tree still holding PR #519's portal commit, so #521's head contained both.
#521 merged first and carried the portal fix; #519 then merged as an **empty
commit** (`131f02b`). Because `deploy-site.yaml` and `web-ci.yml` are path-filtered
on `web/**`, an empty commit matched neither — so the PR whose whole purpose was
fixing two user-facing strings produced no Web CI run and no site deploy, and
confirming the fix was live took a bundle-level audit of what S3 was serving.

Corollaries:
- One topic per PR. A diff spanning `web/` + `infra/` + `lambda/` is usually the
  dirty-tree symptom, not a design.
- After a merge, verify the merge commit is **non-empty** when it should have
  shipped something: `git show --stat <sha>`. An empty diff means another PR
  already carried your work.
- For a `web/**` or `docs/**` change, confirm the deploy workflow actually ran —
  a path-filtered workflow that never triggers looks identical to a passing one.
- Branches are **not** auto-deleted on merge (`delete_branch_on_merge` is off).
  Delete yours after merge, or they accumulate — 66 had to be swept once already.
  The safe test for "is this branch merged?" is **not** `git branch --merged`
  (squash-merge defeats ancestry) and **not** `git diff main <branch>` (a stale
  branch reports main's newer files as deletions). Compare the branch tip to the
  `headRefOid` GitHub recorded for its merged PR.

## Pre-commit Checks
- Run before every commit: `gofmt`, `go vet`, `staticcheck`
- Smoke tests: `go test -short ./...`
- Use pre-commit hook or `make check`

## Testing Workflow
- `make check` — fast: fmt, vet, lint, short tests
- `make test` — full: all unit tests with coverage
- `make integration` — slow: integration/e2e tests
- `make build` — build binary with version
- Run `make check` before every commit

## Project Tracking
**Single Source of Truth: GitHub**
- Track ALL work via GitHub Issues (not ROADMAP.md or other files)
- Use GitHub Projects for planning/status visualization
- Use GitHub Milestones for release grouping
- Use GitHub Labels for categorization
- Close issues via commit message: `Fixes #123`

**Do NOT maintain:**
- ROADMAP.md or similar planning documents
- TODO lists in markdown files
- External tracking spreadsheets
- Status documents

**Rationale:** Maintaining parallel tracking systems leads to synchronization issues. GitHub provides built-in tools for all project management needs.

## Do Not
- Create README, docs, or configs unless asked
- Maintain ROADMAP.md or other project tracking files (use GitHub Issues/Milestones instead)
- Add dependencies without justification
- Use `interface{}` or `any` without reason
- Ignore returned errors
- Use global state
