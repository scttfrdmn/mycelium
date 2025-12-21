#!/bin/bash
# Example: Compare instance types across regions
# Useful for finding alternative instance types with similar specs

INSTANCE_FAMILY="${1:-m7i}"
MIN_VCPU="${2:-4}"
MIN_MEMORY="${3:-16}"

echo "🔍 Finding ${INSTANCE_FAMILY} instances with at least ${MIN_VCPU} vCPUs and ${MIN_MEMORY} GiB RAM"
echo ""

# Search for matching instances
truffle search "${INSTANCE_FAMILY}.*" \
    --min-vcpu "${MIN_VCPU}" \
    --min-memory "${MIN_MEMORY}" \
    --output table

echo ""
echo "📊 Summary by instance type:"
truffle search "${INSTANCE_FAMILY}.*" \
    --min-vcpu "${MIN_VCPU}" \
    --min-memory "${MIN_MEMORY}" \
    --output json 2>/dev/null | \
    jq -r '.[] | .instance_type' | \
    sort | uniq -c | \
    awk '{printf "  %-20s available in %2d regions\n", $2, $1}'

echo ""
echo "💡 Tip: Compare with ${INSTANCE_FAMILY}a.* for AMD alternatives or m8g.* for Graviton4"
