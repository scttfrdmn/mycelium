# spore.host CI runner fleet (orion.local)

The self-hosted GitHub Actions runner fleet that the suite's `Test` and
`E2E Tier 0 (Substrate)` jobs are pinned to (`runs-on: [self-hosted, linux,
orion]`). Six **org-level, ephemeral** runners as a Docker Compose project on
`orion.local` (colima/Docker backend). Versioned here (spore-host#381) — it
previously lived only on the host, so fixes weren't reviewable.

## Layout

| File | Purpose |
|---|---|
| `Dockerfile` | the `spore-runner:latest` image (Ubuntu 24.04 + Go + the actions runner) |
| `entrypoint.sh` | per-cycle: mint a registration token, **self-heal** (`--replace` + clear `_work`/cap go-build cache), register ephemeral, run one job |
| `docker-compose.yml` | the 6-replica fleet (`restart: always`, locally built image) |
| `.env.example` | the one secret: `ACCESS_TOKEN` (PAT with `admin:org`). Real `.env` is host-only, gitignored |
| `boot-recreate.sh` | recreate the fleet (used at boot + manual recovery) |
| `host.spore-ci-runners.plist` | launchd unit that runs `boot-recreate.sh` at host boot |

The live host copy is `orion.local:~/spore-runner-deploy` (compose + `.env`) with
the image build context at `~/spore-runner`. Keep them in sync with this dir.

## Why it kept breaking (spore-host#381)

The runners are ephemeral (one job, then exit) but the compose uses
`restart: always`, which **restarts the same container in place** rather than
recreating it. Two failure modes resulted:

1. **Disk fill → jobs orphaned mid-run.** Each reused container's writable layer
   accumulated `_work` + the Go build cache (~3.5GB) + modules across 45-58
   cycles → ~7GB/container × 6 → the colima volume hit 80%+ → a job mid-run ran
   out of space and the runner died, leaving the GitHub job to time out after
   ~10 min with **no recorded steps** (the signature: it never logged "Set up
   job"). This is what failed spawn#258.
2. **Reboot crash-loop.** After a host reboot, the reused containers' baked
   `.runner` config made `config.sh` refuse ("already configured") and
   `restart: always` looped them forever → all runners offline.

## The durable fix (in this dir)

- **`entrypoint.sh` self-heals:** clears `_work/_temp` + `_work/_actions` and caps
  the go-build cache (wipes if >4GB) at the start of every cycle, so a reused
  container can't grow unbounded; `--replace` re-registers cleanly so a baked
  `.runner` no longer crash-loops.
- **`boot-recreate.sh` + launchd:** at boot, wait for colima then `down
  --remove-orphans && up -d` (recreate, not restart-in-place) and prune offline
  ghost registrations.

## Operations

**Recover now (jobs stuck / disk full / offline ghosts):**
```sh
ssh orion.local
export PATH=/opt/homebrew/bin:$PATH
cd ~/spore-runner-deploy && docker compose down --remove-orphans && docker compose up -d
# verify: 6 containers Up, each log shows "Listening for Jobs"
docker compose logs --tail=40 runner | grep -c "Listening for Jobs"   # → 6
```

**Rebuild the image after changing `Dockerfile`/`entrypoint.sh`:**
```sh
# copy this dir's Dockerfile + entrypoint.sh to ~/spore-runner, then:
cd ~/spore-runner && docker build -t spore-runner:latest .
cd ~/spore-runner-deploy && docker compose down --remove-orphans && docker compose up -d
```

**Install the boot unit (one-time):**
```sh
cp infra/ci-runners/boot-recreate.sh ~/spore-runner-deploy/boot-recreate.sh && chmod +x ~/spore-runner-deploy/boot-recreate.sh
cp infra/ci-runners/host.spore-ci-runners.plist ~/Library/LaunchAgents/com.spore.ci-runners.plist
launchctl load -w ~/Library/LaunchAgents/com.spore.ci-runners.plist
```

**Check health (needs `admin:org`):**
```sh
gh api orgs/spore-host/actions/runners --jq '.total_count, (.runners[]|"\(.name) \(.status)")'
```

## Notes
- Repo-scope queries (`gh api repos/spore-host/spawn/actions/runners`) show **0** —
  these are **org** runners; always query the org scope.
- Disk lives on the colima VM volume `/dev/vdb1`; check with
  `colima ssh -- df -h /mnt/lima-colima`.
- Related: spore-host#345 (Node24 env injection via the entrypoint).
