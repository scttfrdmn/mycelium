#!/usr/bin/env bash
# cleanup-orphan-amis.sh — deregister orphaned spore.host app AMIs and delete
# their backing EBS snapshots, in the AMI's OWNING account (#290 Phase F, #389).
#
# The container catalog (#290) retired the per-app, per-region AMIs. This sweeps
# the leftovers: every self-owned AMI tagged spore:app (or named spore-<app>-*)
# that is NOT the current base AMI and NOT referenced by the catalog. Deregister
# alone leaks the snapshot (~$0.05/GB-mo), so we delete the snapshots too.
#
# DRY-RUN BY DEFAULT — prints what it WOULD delete and exits 0. Set
# SPORE_CLEANUP_APPLY=true to actually deregister + delete.
#
# Usage:
#   ./cleanup-orphan-amis.sh <region> [region...]
#   SPORE_CLEANUP_APPLY=true ./cleanup-orphan-amis.sh us-east-1
#
# Environment:
#   SPORE_KEEP_AMIS   comma-separated AMI IDs to ALWAYS keep (e.g. the live base
#                     AMIs still referenced by catalog base_amis). Safety net.
#   SPORE_CLEANUP_APPLY  "true" to perform deletes (default: dry-run).
#
# Run with credentials for the account that OWNS the AMIs (e.g. the dedicated
# infra account 812107987990, or the beta account 942542972736). Self-owned only.
set -euo pipefail

APPLY="${SPORE_CLEANUP_APPLY:-false}"
KEEP="${SPORE_KEEP_AMIS:-}"

if [[ $# -eq 0 ]]; then
  echo "Usage: $0 <region> [region...]   (dry-run unless SPORE_CLEANUP_APPLY=true)" >&2
  exit 2
fi

# Build a quick lookup of AMI IDs to keep.
declare -A keep_set
IFS=',' read -ra _keep <<< "$KEEP"
for k in "${_keep[@]}"; do
  [[ -n "$k" ]] && keep_set["$k"]=1
done

acct=$(aws sts get-caller-identity --query Account --output text)
echo "==> Owning account: ${acct}   mode: $([[ "$APPLY" == "true" ]] && echo APPLY || echo DRY-RUN)"
[[ -n "$KEEP" ]] && echo "==> Always-keep: ${KEEP}"

total_amis=0
total_snaps=0

for region in "$@"; do
  echo "── ${region} ──────────────────────────────────────────────"

  # Self-owned spore app AMIs: tagged spore:app OR named spore-* (covers older
  # builds that predate the tag). Base AMIs (spore:type=dcv-*-base) are excluded.
  mapfile -t amis < <(aws ec2 describe-images --region "$region" --owners self \
    --filters "Name=name,Values=spore-*" \
    --query 'Images[].{id:ImageId,name:Name,type:Tags[?Key==`spore:type`]|[0].Value}' \
    --output text 2>/dev/null | sort)

  if [[ ${#amis[@]} -eq 0 ]]; then
    echo "  (no self-owned spore-* AMIs)"
    continue
  fi

  while IFS=$'\t' read -r ami_id name ami_type; do
    [[ -z "$ami_id" ]] && continue

    # Skip the shared base AMIs — they're the live catalog targets.
    if [[ "$ami_type" == dcv-*-base ]]; then
      echo "  KEEP (base)   ${ami_id}  ${name}"
      continue
    fi
    if [[ -n "${keep_set[$ami_id]:-}" ]]; then
      echo "  KEEP (listed) ${ami_id}  ${name}"
      continue
    fi

    # Backing snapshots for this AMI.
    mapfile -t snaps < <(aws ec2 describe-images --region "$region" --image-ids "$ami_id" \
      --query 'Images[0].BlockDeviceMappings[].Ebs.SnapshotId' --output text 2>/dev/null | tr '\t' '\n' | grep -v '^None$' || true)

    echo "  ORPHAN        ${ami_id}  ${name}  snaps=[${snaps[*]:-}]"
    total_amis=$((total_amis + 1))
    total_snaps=$((total_snaps + ${#snaps[@]}))

    if [[ "$APPLY" == "true" ]]; then
      echo "    deregister ${ami_id}"
      aws ec2 deregister-image --region "$region" --image-id "$ami_id"
      for s in "${snaps[@]}"; do
        [[ -z "$s" ]] && continue
        echo "    delete snapshot ${s}"
        aws ec2 delete-snapshot --region "$region" --snapshot-id "$s" || \
          echo "    WARN: could not delete ${s} (still in use?)"
      done
    fi
  done < <(printf '%s\n' "${amis[@]}")
done

echo "==> $([[ "$APPLY" == "true" ]] && echo "Deleted" || echo "Would delete") ${total_amis} AMIs + ${total_snaps} snapshots"
[[ "$APPLY" != "true" ]] && echo "    (dry-run; set SPORE_CLEANUP_APPLY=true to apply)"
exit 0
