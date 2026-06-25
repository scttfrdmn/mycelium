#!/bin/bash
# Recreate the spore.host CI runner fleet (spore-host#381). Run at host boot by
# the launchd unit, and safe to run by hand any time the org shows offline ghosts
# or jobs are stuck queued. Idempotent.
#
# Why recreate (not `docker compose restart`): the runners are ephemeral; a fresh
# container re-registers cleanly and starts with empty _work/caches. Restarting
# the pre-reboot containers in place reuses their baked .runner config (crash-loop)
# and their grown writable layers (disk fill).
set -u
export PATH=/opt/homebrew/bin:$PATH
DEPLOY_DIR="${SPORE_RUNNER_DIR:-/Users/scttfrdmn/spore-runner-deploy}"

echo "[$(date -u +%FT%TZ)] spore-ci-runners boot-recreate starting"

# 1. Wait for colima (docker backend) to be up — up to ~3 min after boot.
for i in $(seq 1 36); do
  if colima status >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
    break
  fi
  echo "waiting for colima/docker ($i)…"
  sleep 5
done
if ! docker info >/dev/null 2>&1; then
  echo "colima/docker not up — trying 'colima start'"
  colima start || { echo "FATAL: colima not available"; exit 1; }
fi

# 2. Recreate the fleet (fresh containers; --remove-orphans clears stale ones).
cd "$DEPLOY_DIR" || { echo "FATAL: $DEPLOY_DIR missing"; exit 1; }
docker compose down --remove-orphans || true
docker compose up -d

# 3. Prune offline/ghost org runner registrations the ephemeral fleet left behind
# on unclean shutdowns (harmless but they clutter the org list). Best-effort.
if command -v gh >/dev/null 2>&1; then
  gh api orgs/spore-host/actions/runners --paginate \
    --jq '.runners[] | select(.status=="offline") | .id' 2>/dev/null \
    | while read -r id; do
        [ -n "$id" ] && gh api -X DELETE "orgs/spore-host/actions/runners/$id" >/dev/null 2>&1 \
          && echo "pruned offline runner $id"
      done
fi

echo "[$(date -u +%FT%TZ)] spore-ci-runners boot-recreate done"
