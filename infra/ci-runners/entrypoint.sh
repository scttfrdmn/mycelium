#!/bin/bash
set -e
# Org-scoped ephemeral runner. Mints a fresh registration token from the PAT
# each start, registers, runs one job (ephemeral), then exits and is recreated.

# Unique runner name: "<RUNNER_NAME>-<hostname>". Under docker compose replicas,
# each container has a distinct hostname, so each runner registers uniquely.
RUNNER_BASE="${RUNNER_NAME:-orion}"
RUNNER_FULL="${RUNNER_BASE}-$(hostname)"

# Self-heal disk growth (spore-host#381). The fleet runs `restart: always`, so a
# finished ephemeral container is restarted IN PLACE — its writable layer (and
# _work + the Go build/module caches) persists and grows unbounded across cycles
# (observed: ~7GB/container × 6 → filled the colima volume → jobs orphaned
# mid-run). Reset the per-job workspace and cap the Go cache at the start of every
# cycle so a reused container behaves like a fresh one. Cheap: _work is rebuilt
# per job, and the Go module cache (GOPATH/pkg/mod) is preserved for speed.
cd /home/runner
rm -rf /home/runner/_work/_temp/* 2>/dev/null || true
rm -rf /home/runner/_work/_actions/* 2>/dev/null || true
# Bound the build cache: if it exceeds the cap, wipe it (it repopulates per job).
GOCACHE_DIR="${GOCACHE:-/home/runner/.cache/go-build}"
if [ -d "$GOCACHE_DIR" ]; then
  cache_mb=$(du -sm "$GOCACHE_DIR" 2>/dev/null | awk '{print $1}')
  if [ "${cache_mb:-0}" -gt 4000 ]; then
    echo "go-build cache ${cache_mb}MB > 4000MB cap — clearing (spore-host#381)"
    rm -rf "$GOCACHE_DIR"/* 2>/dev/null || true
  fi
fi

RUNNER_TOKEN=$(curl -fsSL -X POST \
  -H "Authorization: Bearer ${ACCESS_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  https://api.github.com/orgs/${ORG_NAME}/actions/runners/registration-token | jq -r .token)

if [ -z "$RUNNER_TOKEN" ] || [ "$RUNNER_TOKEN" = "null" ]; then
  echo "FATAL: could not mint registration token (check ACCESS_TOKEN / admin:org scope)" >&2
  exit 1
fi

# Inject job-level env for every workflow run on this runner. The runner sources
# /home/runner/.env into each job's environment. FORCE_JAVASCRIPT_ACTIONS_TO_NODE24
# opts bundled Node 20 actions (e.g. github-script inside codecov-action@v5) onto
# Node 24 ahead of GitHub's 2026-06-16 forced cutover — a cross-repo fix with no
# per-repo workflow changes (spore-host#345).
grep -q '^FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=' /home/runner/.env 2>/dev/null \
  || echo 'FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true' >> /home/runner/.env

# Clear any stale LOCAL registration before configuring.
#
# `--replace` (below) resolves the *server-side* name conflict only. The check that
# actually fires on a reused container is local: config.sh refuses outright when
# /home/runner/.runner exists —
#
#   Cannot configure the runner because it is already configured.
#   To reconfigure the runner, run 'config.cmd remove' or './config.sh remove' first.
#
# On a clean exit the `trap cleanup EXIT` below removes the registration, so
# .runner is gone. On an UNCLEAN stop (host reboot, VM killed, OOM) the trap never
# runs, .runner survives, and `restart: always` then restarts that container into an
# unbreakable crash-loop. That is exactly what happened to all 6 runners on
# 2026-08-01 despite --replace being present (#518) — so do the local removal
# ourselves rather than assuming --replace covers it.
if [ -f /home/runner/.runner ]; then
  echo "stale .runner from an unclean stop — removing local registration first (#518)"
  RM_TOKEN=$(curl -fsSL -X POST \
    -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    "https://api.github.com/orgs/${ORG_NAME}/actions/runners/remove-token" | jq -r .token)
  if [ -n "$RM_TOKEN" ] && [ "$RM_TOKEN" != "null" ]; then
    ./config.sh remove --token "$RM_TOKEN" || true
  fi
  # config.sh remove can itself fail against a registration the server already
  # dropped; the local files are what block config.sh, so clear them regardless.
  rm -f /home/runner/.runner /home/runner/.credentials /home/runner/.credentials_rsaparams 2>/dev/null || true
fi

# --replace additionally lets us take over a registration the server still lists
# under this name (e.g. the ghost from a killed container) instead of erroring.
./config.sh \
  --url "https://github.com/${ORG_NAME}" \
  --token "$RUNNER_TOKEN" \
  --name "$RUNNER_FULL" \
  --labels "self-hosted,linux,orion" \
  --work /home/runner/_work \
  --ephemeral \
  --unattended \
  --replace

cleanup() {
  echo "Removing runner registration..."
  TOK=$(curl -fsSL -X POST -H "Authorization: Bearer ${ACCESS_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    https://api.github.com/orgs/${ORG_NAME}/actions/runners/remove-token | jq -r .token)
  ./config.sh remove --token "$TOK" || true
}
trap cleanup EXIT
./run.sh
