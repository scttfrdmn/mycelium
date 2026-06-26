#!/usr/bin/env bash
# share-base-ami.sh — share a spore-dcv-base AMI (and its backing snapshots) with
# the launch account(s), then print the catalog base_amis line.
#
# This is the fix for the #389 root cause: every per-app AMI in the old catalog
# was built in one account and NEVER shared to the account that launches
# instances, so `spawn app launch` failed at RunInstances with AuthFailure. The
# container catalog (#290) has exactly ONE shared base AMI per region, so this
# runs once per region instead of once per app per region.
#
# Usage:
#   ./share-base-ami.sh <ami-id> <region> <launch-account-id> [launch-account-id...]
#   ./share-base-ami.sh ami-0c37fb59c90d1ed3a us-east-1 435415984226
#
# Run with credentials for the AMI's OWNING account (the dedicated infra account
# 812107987990). Idempotent — re-running just re-asserts the launch permissions.
set -euo pipefail

AMI_ID="${1:-}"
REGION="${2:-}"
shift 2 || true
ACCOUNTS=("$@")

if [[ -z "$AMI_ID" || -z "$REGION" || ${#ACCOUNTS[@]} -eq 0 ]]; then
  echo "Usage: $0 <ami-id> <region> <launch-account-id> [more-accounts...]" >&2
  exit 2
fi

echo "==> Confirming ${AMI_ID} exists in ${REGION} (owning account)"
aws ec2 describe-images --region "$REGION" --image-ids "$AMI_ID" \
  --query 'Images[0].[ImageId,Name,State]' --output text

# Build the LaunchPermission Add list.
ADD_JSON="$(printf '{"UserId":"%s"},' "${ACCOUNTS[@]}")"
ADD_JSON="[${ADD_JSON%,}]"

echo "==> Granting launch permission to: ${ACCOUNTS[*]}"
aws ec2 modify-image-attribute --region "$REGION" --image-id "$AMI_ID" \
  --launch-permission "{\"Add\":${ADD_JSON}}"

# The instance can't boot unless the backing EBS snapshots are also shared.
echo "==> Sharing backing snapshots"
SNAP_IDS=$(aws ec2 describe-images --region "$REGION" --image-ids "$AMI_ID" \
  --query 'Images[0].BlockDeviceMappings[].Ebs.SnapshotId' --output text)
for snap in $SNAP_IDS; do
  [[ -z "$snap" || "$snap" == "None" ]] && continue
  echo "    ${snap}"
  aws ec2 modify-snapshot-attribute --region "$REGION" --snapshot-id "$snap" \
    --attribute createVolumePermission --operation-type add \
    --user-ids "${ACCOUNTS[@]}"
done

echo "==> Verifying launch permission"
aws ec2 describe-image-attribute --region "$REGION" --image-id "$AMI_ID" \
  --attribute launchPermission --query 'LaunchPermissions' --output json

echo "==> Done. Catalog line for libs/catalog/catalog.yaml base_amis:"
echo "      ${REGION}: ${AMI_ID}"
