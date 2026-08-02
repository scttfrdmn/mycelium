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

**Rebuild the image after changing `Dockerfile`/`entrypoint.sh`:**
```sh
# from a clone of this repo, with the fleet IDLE (a recreate kills running jobs):
scp infra/ci-runners/{Dockerfile,entrypoint.sh} orion.local:~/spore-runner/
ssh orion.local 'export PATH=/opt/homebrew/bin:$PATH
  cd ~/spore-runner && docker build -t spore-runner:latest .
  bash ~/spore-runner-deploy/boot-recreate.sh'
```
A change to `entrypoint.sh` is **baked into the image** — copying it to the host is
not enough, and neither is a recreate. Rebuild, or the fleet keeps running the old
entrypoint. Confirm what's actually live:
```sh
docker run --rm --entrypoint /bin/bash spore-runner:latest -c 'md5sum /home/runner/entrypoint.sh'
md5 -q infra/ci-runners/entrypoint.sh   # must match
```

**Check for host drift** (this dir is the source of truth; the host copies are
deployed by hand, so they diverge silently):
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
