#!/bin/bash
# Example: CI/CD validation script
# Validates that required instance types are available before deployment

set -e

REQUIRED_INSTANCE_TYPE="${INSTANCE_TYPE:-t3a.medium}"
TARGET_REGION="${AWS_REGION:-us-east-1}"
MIN_AZS="${MIN_AZS:-2}"

echo "🔍 Validating deployment prerequisites..."
echo "   Instance Type: ${REQUIRED_INSTANCE_TYPE}"
echo "   Region: ${TARGET_REGION}"
echo "   Minimum AZs: ${MIN_AZS}"
echo ""

# Check if instance type is available
result=$(truffle search "${REQUIRED_INSTANCE_TYPE}" \
    --regions "${TARGET_REGION}" \
    --include-azs \
    --output json 2>/dev/null)

if [ -z "$result" ] || [ "$(echo "$result" | jq '. | length')" -eq 0 ]; then
    echo "❌ VALIDATION FAILED: ${REQUIRED_INSTANCE_TYPE} not available in ${TARGET_REGION}"
    exit 1
fi

# Check number of availability zones
az_count=$(echo "$result" | jq -r '.[0].availability_zones | length')

if [ "${az_count}" -lt "${MIN_AZS}" ]; then
    echo "❌ VALIDATION FAILED: Only ${az_count} AZs available, need at least ${MIN_AZS}"
    echo "   Available AZs: $(echo "$result" | jq -r '.[0].availability_zones | join(", ")')"
    exit 1
fi

# Display validation success
vcpus=$(echo "$result" | jq -r '.[0].vcpus')
memory=$(echo "$result" | jq -r '.[0].memory_mib / 1024')
azs=$(echo "$result" | jq -r '.[0].availability_zones | join(", ")')

echo "✅ VALIDATION PASSED"
echo ""
echo "   Instance specs:"
echo "     - vCPUs: ${vcpus}"
echo "     - Memory: ${memory} GiB"
echo "     - Available AZs (${az_count}): ${azs}"
echo ""
echo "✨ Deployment can proceed!"

exit 0
