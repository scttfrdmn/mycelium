#!/bin/bash
# Example: Plan multi-region deployment
# This script checks if your desired instance type is available in target regions

INSTANCE_TYPE="m7i.xlarge"
TARGET_REGIONS=("us-east-1" "us-west-2" "eu-west-1" "ap-southeast-1")

echo "🔍 Checking availability of ${INSTANCE_TYPE} across regions..."
echo ""

for region in "${TARGET_REGIONS[@]}"; do
    echo -n "Checking ${region}... "
    
    result=$(truffle search "${INSTANCE_TYPE}" --regions "${region}" --output json 2>/dev/null)
    
    if [ $(echo "$result" | jq '. | length') -gt 0 ]; then
        vcpus=$(echo "$result" | jq -r '.[0].vcpus')
        memory=$(echo "$result" | jq -r '.[0].memory_mib / 1024')
        echo "✅ Available (${vcpus} vCPUs, ${memory} GiB RAM)"
    else
        echo "❌ Not available"
    fi
done

echo ""
echo "💡 Tip: Use --include-azs flag to check AZ-level availability"
