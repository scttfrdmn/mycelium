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

# 1b. Still no docker: escalate. `colima start` alone is NOT enough — a VM left
# `Broken` by an unclean shutdown (seen 2026-08-01: the vz driver process survives
# but its host agent doesn't, so lima reports "vz driver is running but host agent
# is not") makes `colima start` exit 1 on inspection without ever trying to boot:
#
#   level=fatal msg="errors inspecting instance: [vz driver is running but host agent is not]"
#
# The fleet then stayed down ~5.5h because this script exited here (#518).
# `colima stop --force` reaps the orphaned driver and moves Broken → Stopped, after
# which a normal start works. Escalate on ANY failed start, not on a parsed status
# string: the goal is never to exit while an untried recovery remains.
if ! docker info >/dev/null 2>&1; then
  echo "colima/docker not up — trying 'colima start'"
  if ! colima start; then
    echo "'colima start' failed — VM is likely Broken; forcing stop then restarting"
    colima list 2>&1 || true
    colima stop --force || true
    sleep 3
    # A failed shutdown can also leave the lima disk locked; releasing it is
    # harmless when it's already unlocked ("Ignoring unlocked disk").
    LIMA_HOME="$HOME/.colima/_lima" limactl disk unlock colima >/dev/null 2>&1 || true
    colima start || { echo "FATAL: colima not available even after stop --force"; exit 1; }
  fi
fi

if ! docker info >/dev/null 2>&1; then
  echo "FATAL: colima reports started but docker is not usable"; exit 1
fi

# 2. Recreate the fleet (fresh containers; --remove-orphans clears stale ones).
cd "$DEPLOY_DIR" || { echo "FATAL: $DEPLOY_DIR missing"; exit 1; }
docker compose down --remove-orphans || true
docker compose up -d

# 3. Prune offline/ghost org runner registrations the ephemeral fleet left behind
# on unclean shutdowns (harmless but they clutter the org list, and they make
# "is the fleet up?" unanswerable at a glance).
#
# This used to be gated on `command -v gh`. `gh` is NOT installed on orion, so the
# whole block silently no-opped on every boot — it read as a safety net without
# being one (#518). Use curl + jq (both present: /usr/bin/curl, /usr/bin/jq) with
# the ACCESS_TOKEN already in .env — the same token entrypoint.sh mints
# registration tokens with, so this needs no new credential.
ORG="${ORG_NAME:-spore-host}"
if [ -f "$DEPLOY_DIR/.env" ]; then
  # Read ACCESS_TOKEN without exporting the whole .env or logging the value.
  ACCESS_TOKEN=$(sed -n 's/^ACCESS_TOKEN=//p' "$DEPLOY_DIR/.env" | head -1)
fi
if [ -z "${ACCESS_TOKEN:-}" ]; then
  echo "WARN: no ACCESS_TOKEN available — skipping offline-runner prune"
elif ! command -v jq >/dev/null 2>&1; then
  echo "WARN: jq not found — skipping offline-runner prune"
else
  gh_api() { # gh_api <method> <path>
    curl -fsS -X "$1" \
      -H "Authorization: Bearer ${ACCESS_TOKEN}" \
      -H "Accept: application/vnd.github+json" \
      "https://api.github.com/$2"
  }
  if ! runners_json=$(gh_api GET "orgs/${ORG}/actions/runners?per_page=100"); then
    echo "WARN: could not list org runners (check ACCESS_TOKEN admin:org scope) — prune skipped"
  else
    pruned=0
    # Only offline AND not busy: never delete a registration mid-job.
    for id in $(printf '%s' "$runners_json" \
                | jq -r '.runners[] | select(.status=="offline" and .busy==false) | .id'); do
      if gh_api DELETE "orgs/${ORG}/actions/runners/${id}" >/dev/null; then
        echo "pruned offline runner ${id}"
        pruned=$((pruned + 1))
      else
        echo "WARN: failed to prune offline runner ${id}"
      fi
    done
    echo "offline-runner prune: ${pruned} removed"
    online=$(printf '%s' "$runners_json" | jq -r '[.runners[]|select(.status=="online")]|length')
    echo "org runners online before prune: ${online}"
  fi
fi

echo "[$(date -u +%FT%TZ)] spore-ci-runners boot-recreate done"
