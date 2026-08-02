#!/bin/bash
# Sync the live orion fleet to this repo's infra/ci-runners/ (spore-host#522).
#
# Why this exists: the versioned copies here and the live host copies
# (~/spore-runner, ~/spore-runner-deploy) were deployed BY HAND, so they drifted
# silently. Reviewing a fix here proved nothing about what the fleet was actually
# running — and #518's defect 3 shipped a fix (`--replace`) that was live in the
# image yet didn't work, which is exactly the class of thing hand-deploys hide.
#
# What it does, in order:
#   1. fetch + hard-reset a bare-ish cache clone of main (no local edits kept)
#   2. copy scripts into place ONLY if their content changed
#   3. rebuild the runner image if the build context changed (Dockerfile/entrypoint)
#   4. recreate the fleet if anything changed AND the fleet is idle
# Idempotent: a no-drift run makes no changes, touches no containers, and exits 0.
#
# Safety: never recreates while a runner is busy (that would kill a live CI job).
# If busy, it reports what's pending and exits 0 — the next run picks it up. Run
# from cron/launchd; see README.
set -u
export PATH=/opt/homebrew/bin:$PATH

REPO_URL="${SPORE_REPO_URL:-https://github.com/spore-host/spore-host.git}"
CACHE_DIR="${SPORE_REPO_CACHE:-$HOME/.cache/spore-ci-runners-src}"
BUILD_DIR="${SPORE_RUNNER_BUILD_DIR:-$HOME/spore-runner}"
DEPLOY_DIR="${SPORE_RUNNER_DIR:-$HOME/spore-runner-deploy}"
SRC_SUBDIR="infra/ci-runners"
FORCE="${SPORE_SYNC_FORCE:-0}"      # 1 = recreate even if nothing changed
DRY_RUN="${SPORE_SYNC_DRY_RUN:-0}"  # 1 = report drift, change nothing

log() { echo "[$(date -u +%FT%TZ)] sync-from-git: $*"; }

# --- 1. Refresh the source of truth ------------------------------------------
if [ -d "$CACHE_DIR/.git" ]; then
  git -C "$CACHE_DIR" fetch --quiet --depth 1 origin main || {
    log "FATAL: git fetch failed"; exit 1; }
  git -C "$CACHE_DIR" reset --quiet --hard FETCH_HEAD || {
    log "FATAL: git reset failed"; exit 1; }
else
  rm -rf "$CACHE_DIR"
  git clone --quiet --depth 1 --branch main "$REPO_URL" "$CACHE_DIR" || {
    log "FATAL: git clone failed"; exit 1; }
fi
SRC="$CACHE_DIR/$SRC_SUBDIR"
[ -d "$SRC" ] || { log "FATAL: $SRC missing in the clone"; exit 1; }
HEAD_SHA=$(git -C "$CACHE_DIR" rev-parse --short HEAD)
log "source at $HEAD_SHA"

# Heartbeat for the CI drift gate. The workflow can't read the host FS from inside
# a container, so it checks this instead: a missing or stale manifest means THIS
# script stopped running, which is the failure that lets host-side drift pile up
# unnoticed. Written on every successful fetch, including no-drift runs — the point
# is to prove liveness, not to record changes. The entrypoint copies it into each
# container (see entrypoint.sh).
write_manifest() {
  [ "$DRY_RUN" = "1" ] && return 0
  mkdir -p "$DEPLOY_DIR"
  cat > "$DEPLOY_DIR/.sync-manifest" <<EOF
sha=$HEAD_SHA
synced_at=$(date -u +%FT%TZ)
EOF
}
write_manifest

# --- 2. Deploy changed files -------------------------------------------------
# hash <file> — content hash, or "-" when absent. macOS `md5 -q`.
hash_of() { [ -f "$1" ] && md5 -q "$1" || echo "-"; }

changed_build=0   # Dockerfile/entrypoint.sh → needs an image rebuild
changed_any=0

deploy() { # deploy <src-name> <dest-path> <is-build-input>
  local src="$SRC/$1" dest="$2" is_build="$3"
  [ -f "$src" ] || { log "WARN: $1 missing in repo — skipped"; return 0; }
  if [ "$(hash_of "$src")" = "$(hash_of "$dest")" ]; then
    return 0
  fi
  log "DRIFT: $1 differs from $dest"
  changed_any=1
  [ "$is_build" = "1" ] && changed_build=1
  if [ "$DRY_RUN" = "1" ]; then
    log "  (dry run — not copying)"
    return 0
  fi
  mkdir -p "$(dirname "$dest")"
  cp "$src" "$dest" || { log "FATAL: copy $1 failed"; exit 1; }
  case "$dest" in *.sh) chmod +x "$dest";; esac
  log "  updated $dest"
}

deploy Dockerfile        "$BUILD_DIR/Dockerfile"          1
deploy entrypoint.sh     "$BUILD_DIR/entrypoint.sh"       1
deploy docker-compose.yml "$DEPLOY_DIR/docker-compose.yml" 0
deploy boot-recreate.sh  "$DEPLOY_DIR/boot-recreate.sh"   0
deploy sync-from-git.sh  "$DEPLOY_DIR/sync-from-git.sh"   0

# The image can also drift from an up-to-date build context — e.g. someone edited
# entrypoint.sh on the host and never rebuilt, or a rebuild failed half-done. The
# baked copy is what actually runs, so compare against THAT, not just the files.
if docker image inspect spore-runner:latest >/dev/null 2>&1; then
  baked=$(docker run --rm --entrypoint /bin/bash spore-runner:latest \
            -c 'md5sum /home/runner/entrypoint.sh' 2>/dev/null | awk '{print $1}')
  want=$(hash_of "$SRC/entrypoint.sh")
  if [ -n "$baked" ] && [ "$baked" != "$want" ]; then
    log "DRIFT: image entrypoint ($baked) != repo ($want) — rebuild needed"
    changed_any=1; changed_build=1
  fi
else
  log "DRIFT: spore-runner:latest image missing — build needed"
  changed_any=1; changed_build=1
fi

if [ "$changed_any" = "0" ] && [ "$FORCE" != "1" ]; then
  log "no drift — nothing to do"
  exit 0
fi
if [ "$DRY_RUN" = "1" ]; then
  log "dry run complete — drift found, no changes made"
  exit 0
fi

# --- 3. Never disturb a running job -----------------------------------------
# A rebuild + recreate tears down containers. Doing that mid-job kills the CI run
# (which shows up as a job that times out with no recorded steps — the #381
# signature). Defer instead; the next scheduled run applies it.
busy=0
for c in $(docker ps --filter "name=spore-runner-deploy-runner" --format '{{.Names}}' 2>/dev/null); do
  if docker exec "$c" sh -c 'ps ax 2>/dev/null | grep -q "[R]unner.Worker"' 2>/dev/null; then
    busy=$((busy + 1))
  fi
done
if [ "$busy" -gt 0 ]; then
  log "$busy runner(s) busy — deferring rebuild/recreate to the next run (files are staged)"
  exit 0
fi

# --- 4. Rebuild + recreate ---------------------------------------------------
if [ "$changed_build" = "1" ]; then
  log "rebuilding spore-runner:latest"
  ( cd "$BUILD_DIR" && docker build -q -t spore-runner:latest . ) >/dev/null || {
    log "FATAL: docker build failed — fleet left on the OLD image (still serving)"; exit 1; }
  log "rebuild done"
fi

log "recreating the fleet"
bash "$DEPLOY_DIR/boot-recreate.sh" || { log "FATAL: boot-recreate failed"; exit 1; }
log "sync complete"
