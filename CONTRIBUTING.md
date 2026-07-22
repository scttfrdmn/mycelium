# Contributing to spore.host

Thanks for your interest in improving spore.host. This repo (`spore-host`) is the
umbrella: it holds the documentation site, the hosted API/dashboard, deployment
automation, and this project-wide guidance. The individual tools live in their own
repositories.

## Where things live

The fastest way to report anything — a bug, a docs problem, a question — is
[Discord](https://discord.gg/2deGRFCW); maintainers triage from there and open the
tracking issue in the right repo. If you're a contributor sending a PR, here's
where each tool's code and issues live:

| Area | Repo |
|--------------|------|
| Docs, website, dashboard, or API | [spore-host](https://github.com/spore-host/spore-host) (this repo) |
| File a bug or request a feature in **spawn** (launch/lifecycle) | [spawn](https://github.com/spore-host/spawn) |
| …in **truffle** (discovery/pricing/quotas) | [truffle](https://github.com/spore-host/truffle) |
| …in **lagotto** (capacity watching) | [lagotto](https://github.com/spore-host/lagotto) |
| …in the **MCP server** | [spore-host-mcp](https://github.com/spore-host/spore-host-mcp) |
| Add or fix an official **plugin** | [spore-plugins](https://github.com/spore-host/spore-plugins) |
| A **workflow adapter** (Nextflow/Snakemake/CWL/WDL/Airflow) | its own `*-spawn` / `*-executor-plugin-spawn` repo |

Not sure which repo, or don't have a GitHub account?
[Discord](https://discord.gg/2deGRFCW) is the place — we'll route it from there.

## Getting help vs. reporting

- **Questions, usage help, bugs, and problems:** start in
  [Discord](https://discord.gg/2deGRFCW). It's open to everyone (no GitHub account
  needed), it's where maintainers triage, and it's faster than an issue. This is
  the default channel — use it whenever you're unsure.
- **Contributing a fix:** if you're opening a PR, file or reference an issue in the
  relevant tool repo so the change is tracked (issue templates are provided in each
  repo for contributors with write access; otherwise raise it in Discord first and
  we'll open the tracking issue).
- **Security vulnerability:** do **not** report it publicly — use the private
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
- **Versioning:** [SemVer 2.0.0](https://semver.org) + a Keep-a-Changelog
  `CHANGELOG.md`; update `## [Unreleased]` in the **same PR** as any user-facing change.
- **Docs:** the site is VitePress under `docs/`. `cd docs && npm install && npm run
  docs:dev` to preview; `npm run build` must pass. CLI references are generated —
  don't hand-edit generated pages.

## Cost safety (important)

Several tools launch **real, billable** AWS resources. Any test that touches AWS
must set a TTL, terminate explicitly when done, and be independently leak-checked
(no orphaned instances). Never commit changes that could leave resources running.

## Code of Conduct

By participating you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

## License

Contributions are accepted under the project's [Apache 2.0 license](LICENSE).
