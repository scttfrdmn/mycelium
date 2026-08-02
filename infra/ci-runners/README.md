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
| `entrypoint.sh` | per-cycle: mint a registration token, **self-heal** (clear a stale local `.runner`, `--replace`, clear `_work`/cap go-build cache), register ephemeral, run one job |
| `docker-compose.yml` | the 6-replica fleet (`restart: always`, locally built image) |
| `.env.example` | the one secret: `ACCESS_TOKEN` (PAT with `admin:org`). Real `.env` is host-only, gitignored |
| `boot-recreate.sh` | recreate the fleet (used at boot + manual recovery) |
| `host.spore-ci-runners.plist` | launchd unit that runs `boot-recreate.sh` at host boot |
| `sync-from-git.sh` | reconcile the live host + image to this dir, hourly (spore-host#522) |
| `host.spore-ci-sync.plist` | launchd unit that runs `sync-from-git.sh` hourly and at boot |

The live host copy is `orion.local:~/spore-runner-deploy` (compose + `.env`) with
the image build context at `~/spore-runner`. `sync-from-git.sh` keeps both matching
this dir automatically — see [Automatic sync](#automatic-sync-spore-host522). Merge
to `main` and it converges; don't `scp` fixes onto the host.

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
3. **`Broken` colima VM after reboot** (added 2026-08-01, spore-host#518). See
   below — this one kept the whole fleet down ~5.5h.

## The durable fix (in this dir)

- **`entrypoint.sh` self-heals:** clears `_work/_temp` + `_work/_actions` and caps
  the go-build cache (wipes if >4GB) at the start of every cycle, so a reused
  container can't grow unbounded; and it **removes a stale local `.runner`**
  before `config.sh` runs, so an unclean stop no longer crash-loops.
- **`boot-recreate.sh` + launchd:** at boot, wait for colima (escalating to
  `stop --force` if the VM is `Broken`) then `down --remove-orphans && up -d`
  (recreate, not restart-in-place) and prune offline ghost registrations.

## What `--replace` does NOT fix (spore-host#518)

`--replace` was originally added believing it stopped the reboot crash-loop. **It
doesn't.** On 2026-08-01 all 6 runners crash-looped on exactly
`Cannot configure the runner because it is already configured` *with `--replace`
present in the running image*.

`--replace` resolves the **server-side** name conflict. The check that fires is
**local**: `config.sh` refuses whenever `/home/runner/.runner` exists. On a clean
exit the `trap cleanup EXIT` removes the registration and that file; on an unclean
stop (host reboot, VM killed) the trap never runs, the file survives, and
`restart: always` restarts the container into an unbreakable loop. So
`entrypoint.sh` now explicitly removes the local registration first (`config.sh
remove` when a token can be minted, plus `rm -f .runner .credentials*`
unconditionally, since the local files are what block `config.sh`).

## The `Broken` colima VM (spore-host#518)

Distinct from the disk-full mode — **check the host disk first; if it has room,
this is your failure.** A vz VM whose prior shutdown died (`fatal: vz:
CanRequestStop is not supported`) comes back as lima state `Broken`, not
`Stopped`: the driver process is running but its host agent isn't.

`colima start` **cannot** recover that — it exits 1 during inspection
(`errors inspecting instance: [vz driver is running but host agent is not]`)
without attempting a boot. `boot-recreate.sh` used to stop there and exit 1, so
the fleet stayed down until someone noticed. It now escalates automatically:

```sh
colima stop --force   # Broken → Stopped (also reaps the orphaned vz driver)
colima start          # now succeeds
```

Diagnose with `colima list` (→ `Broken`) or
`LIMA_HOME=$HOME/.colima/_lima limactl list`.

## Operations

**Recover now (jobs stuck / disk full / offline ghosts / after a reboot):**
```sh
ssh orion.local
export PATH=/opt/homebrew/bin:$PATH
bash ~/spore-runner-deploy/boot-recreate.sh   # idempotent; handles a Broken VM too
# verify: 6 containers Up, each log shows "Listening for Jobs"
cd ~/spore-runner-deploy && docker compose logs --tail=40 runner | grep -c "Listening for Jobs"   # → 6
```
Prefer the script over a bare `docker compose up -d`: that **restarts** the stale
pre-reboot containers, which crash-loop. Recreating is what clears them.

**Ship a change to `Dockerfile`/`entrypoint.sh`:** merge it to `main` — the hourly
sync (below) deploys, rebuilds, and recreates on its own. To apply it now rather
than within the hour:
```sh
ssh orion.local 'export PATH=/opt/homebrew/bin:$PATH
  bash ~/spore-runner-deploy/sync-from-git.sh'
```
A change to `entrypoint.sh` is **baked into the image** — copying it to the host is
not enough, and neither is a recreate. `sync-from-git.sh` handles that (it compares
the *baked* copy, not just the host file), but if you build by hand, rebuild.
Confirm what's actually live:
```sh
docker run --rm --entrypoint /bin/bash spore-runner:latest -c 'md5sum /home/runner/entrypoint.sh'
md5 -q infra/ci-runners/entrypoint.sh   # must match
```

## Automatic sync (spore-host#522)

`sync-from-git.sh` makes this directory the fleet's actual definition instead of a
document about it. launchd runs it hourly (and at boot): fetch `main` into a cache
clone, copy only files whose content changed, rebuild when the build context
changed **or** when the baked entrypoint stops matching the repo, then recreate —
but **only when no runner is busy**. A no-drift run touches nothing and exits 0.

It defers rather than interrupts: recreating mid-job kills that CI run, which
surfaces as a job that times out with no recorded steps (the #381 signature). The
next hourly run applies the staged change.

`.github/workflows/ci-runner-drift.yml` is the other half, and it runs *on the
fleet* on purpose — the only way to know what a runner runs is to ask a runner. It
hashes its own baked `/home/runner/entrypoint.sh` against the repo, and separately
checks the `.sync-manifest` heartbeat (bind-mounted `:ro`, so a job can't forge its
own proof of freshness) to catch the sync itself having died. It can't hash
`docker-compose.yml` or `boot-recreate.sh` — a container can't read the host FS —
so it verifies the reconciler is alive and lets the reconciler cover the content.

**Bootstrap (one-time, chicken-and-egg — the script deploys itself, so the first
copy is placed by hand):**
```sh
scp infra/ci-runners/sync-from-git.sh orion.local:~/spore-runner-deploy/
ssh orion.local 'chmod +x ~/spore-runner-deploy/sync-from-git.sh
  # Create the manifest BEFORE the fleet is recreated: Docker creates a
  # DIRECTORY at a bind-mount path whose host file is missing, and a dir there
  # makes the drift gate fail. The first sync run overwrites this.
  : > ~/spore-runner-deploy/.sync-manifest'
scp infra/ci-runners/host.spore-ci-sync.plist orion.local:~/Library/LaunchAgents/com.spore.ci-sync.plist
ssh orion.local 'launchctl load -w ~/Library/LaunchAgents/com.spore.ci-sync.plist'
```
The unit runs at load, so that last command performs the first real sync. Verify:
```sh
ssh orion.local 'tail -40 /tmp/spore-ci-sync.log'
```

**Preview / force / inspect:**
```sh
ssh orion.local 'SPORE_SYNC_DRY_RUN=1 bash ~/spore-runner-deploy/sync-from-git.sh'  # report drift, change nothing
ssh orion.local 'SPORE_SYNC_FORCE=1   bash ~/spore-runner-deploy/sync-from-git.sh'  # recreate even with no drift
ssh orion.local 'tail -40 /tmp/spore-ci-sync.log; cat ~/spore-runner-deploy/.sync-manifest'
```

**Manual drift check** (still useful when the sync is what you suspect):
```sh
for f in entrypoint.sh Dockerfile; do
  ssh orion.local "md5 -q ~/spore-runner/$f" ; md5 -q "infra/ci-runners/$f"
done
ssh orion.local 'md5 -q ~/spore-runner-deploy/boot-recreate.sh'; md5 -q infra/ci-runners/boot-recreate.sh
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
- **`gh` is not installed on orion.** Anything in these scripts gated on
  `command -v gh` silently no-ops there — that's how the ghost-prune went
  unnoticed as dead code for weeks (#518). Use `curl` + `jq` (both at
  `/usr/bin/`) with the `ACCESS_TOKEN` from `.env`, which is already the token
  `entrypoint.sh` mints registration tokens with. Run `gh`-based health checks
  from a dev host instead.
- Repo-scope queries (`gh api repos/spore-host/spawn/actions/runners`) show **0** —
  these are **org** runners; always query the org scope.
- Disk lives on the colima VM volume `/dev/vdb1`; check with
  `colima ssh -- df -h /mnt/lima-colima`.
- Related: spore-host#345 (Node24 env injection via the entrypoint).
