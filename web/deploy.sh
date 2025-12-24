#!/bin/bash
# Deploy spore.host landing page to S3 + CloudFront

set -e  # Exit on error

# Configuration
BUCKET_NAME="spore-host-website"
REGION="us-east-1"
DOMAIN="spore.host"
AWS_PROFILE="${AWS_PROFILE:-default}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   Spore.host Deployment Script       ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo -e "${RED}Error: AWS CLI is not installed${NC}"
    echo "Install: https://aws.amazon.com/cli/"
    exit 1
fi

# Check AWS credentials
echo -e "${BLUE}→${NC} Checking AWS credentials (profile: ${AWS_PROFILE})..."
if ! aws sts get-caller-identity --profile "$AWS_PROFILE" &> /dev/null; then
    echo -e "${RED}Error: AWS credentials not configured${NC}"
    echo "Run: aws configure --profile $AWS_PROFILE"
    exit 1
fi
echo -e "${GREEN}✓${NC} AWS credentials validated"

# Step 1: Create S3 bucket if it doesn't exist
echo ""
echo -e "${BLUE}→${NC} Checking S3 bucket..."
if aws s3 ls "s3://$BUCKET_NAME" --profile "$AWS_PROFILE" 2>/dev/null; then
    echo -e "${GREEN}✓${NC} Bucket exists: $BUCKET_NAME"
else
    echo -e "${YELLOW}!${NC} Creating bucket: $BUCKET_NAME"
    aws s3 mb "s3://$BUCKET_NAME" \
        --region "$REGION" \
        --profile "$AWS_PROFILE"
    echo -e "${GREEN}✓${NC} Bucket created"
fi

# Step 2: Configure bucket for static website hosting
echo ""
echo -e "${BLUE}→${NC} Configuring static website hosting..."
aws s3 website "s3://$BUCKET_NAME" \
    --index-document index.html \
    --error-document index.html \
    --profile "$AWS_PROFILE"
echo -e "${GREEN}✓${NC} Website hosting enabled"

# Step 3: Set bucket policy for public read
echo ""
echo -e "${BLUE}→${NC} Setting bucket policy for public access..."
cat > /tmp/bucket-policy.json <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "PublicReadGetObject",
            "Effect": "Allow",
            "Principal": "*",
            "Action": "s3:GetObject",
            "Resource": "arn:aws:s3:::$BUCKET_NAME/*"
        }
    ]
}
EOF
aws s3api put-bucket-policy \
    --bucket "$BUCKET_NAME" \
    --policy file:///tmp/bucket-policy.json \
    --profile "$AWS_PROFILE"
rm /tmp/bucket-policy.json
echo -e "${GREEN}✓${NC} Bucket policy set"

# Step 4: Upload website files
echo ""
echo -e "${BLUE}→${NC} Uploading website files..."
aws s3 sync . "s3://$BUCKET_NAME/" \
    --delete \
    --exclude ".git/*" \
    --exclude ".DS_Store" \
    --exclude "*.sh" \
    --exclude "README.md" \
    --cache-control "max-age=3600" \
    --profile "$AWS_PROFILE"
echo -e "${GREEN}✓${NC} Files uploaded"

# Step 5: Check for CloudFront distribution
echo ""
echo -e "${BLUE}→${NC} Checking CloudFront distribution..."
DISTRIBUTION_ID=$(aws cloudfront list-distributions \
    --profile "$AWS_PROFILE" \
    --query "DistributionList.Items[?Origins.Items[0].DomainName=='$BUCKET_NAME.s3.amazonaws.com'].Id" \
    --output text 2>/dev/null || echo "")

if [ -z "$DISTRIBUTION_ID" ]; then
    echo -e "${YELLOW}!${NC} No CloudFront distribution found"
    echo ""
    echo -e "${BLUE}To create CloudFront distribution:${NC}"
    echo "1. Go to: https://console.aws.amazon.com/cloudfront/"
    echo "2. Create Distribution"
    echo "3. Origin domain: $BUCKET_NAME.s3-website-$REGION.amazonaws.com"
    echo "4. Viewer protocol policy: Redirect HTTP to HTTPS"
    echo "5. Alternate domain name (CNAME): $DOMAIN"
    echo "6. Custom SSL certificate: Request certificate via ACM"
    echo ""
    echo -e "${BLUE}Then update Route53:${NC}"
    echo "• Create A record for $DOMAIN"
    echo "• Type: Alias"
    echo "• Alias target: CloudFront distribution"
else
    echo -e "${GREEN}✓${NC} CloudFront distribution found: $DISTRIBUTION_ID"

    # Invalidate CloudFront cache
    echo ""
    echo -e "${BLUE}→${NC} Invalidating CloudFront cache..."
    INVALIDATION_ID=$(aws cloudfront create-invalidation \
        --distribution-id "$DISTRIBUTION_ID" \
        --paths "/*" \
        --profile "$AWS_PROFILE" \
        --query 'Invalidation.Id' \
        --output text)
    echo -e "${GREEN}✓${NC} Cache invalidation created: $INVALIDATION_ID"
fi

# Step 6: Display results
echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        Deployment Complete! 🎉         ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}S3 Website URL:${NC}"
echo "http://$BUCKET_NAME.s3-website-$REGION.amazonaws.com"
echo ""

if [ -n "$DISTRIBUTION_ID" ]; then
    CLOUDFRONT_DOMAIN=$(aws cloudfront get-distribution \
        --id "$DISTRIBUTION_ID" \
        --profile "$AWS_PROFILE" \
        --query 'Distribution.DomainName' \
        --output text)
    echo -e "${BLUE}CloudFront URL:${NC}"
    echo "https://$CLOUDFRONT_DOMAIN"
    echo ""
    echo -e "${BLUE}Custom Domain:${NC}"
    echo "https://$DOMAIN"
else
    echo -e "${YELLOW}Note:${NC} Set up CloudFront for HTTPS and better performance"
fi

echo ""
echo -e "${BLUE}Next steps:${NC}"
echo "• Test the site in your browser"
echo "• Configure CloudFront if not already set up"
echo "• Update Route53 DNS to point to CloudFront"
echo "• Request SSL certificate via ACM for HTTPS"
echo ""
