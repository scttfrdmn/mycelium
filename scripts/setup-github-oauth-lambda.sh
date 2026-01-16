#!/bin/bash
set -euo pipefail

# Setup GitHub OAuth Lambda Bridge
# Creates Lambda function and API Gateway to handle GitHub OAuth callbacks

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LAMBDA_DIR="$PROJECT_ROOT/spawn/lambda/github-oauth"
CONFIG_FILE="$PROJECT_ROOT/config/oauth-credentials.json"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║   GitHub OAuth Lambda Setup           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""

# Check AWS profile
if [[ -z "${AWS_PROFILE:-}" ]]; then
    echo -e "${YELLOW}→${NC} AWS_PROFILE not set. Using default credentials..."
else
    echo -e "${BLUE}→${NC} Using AWS profile: ${AWS_PROFILE}"
fi

# Verify AWS credentials
echo -e "${BLUE}→${NC} Checking AWS credentials..."
if ! aws sts get-caller-identity &>/dev/null; then
    echo -e "${RED}✗${NC} Failed to authenticate with AWS"
    exit 1
fi
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
echo -e "${GREEN}✓${NC} AWS credentials validated (Account: $ACCOUNT_ID)"

# Read GitHub OAuth credentials
echo -e "${BLUE}→${NC} Reading GitHub OAuth credentials..."
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo -e "${RED}✗${NC} Config file not found: $CONFIG_FILE"
    exit 1
fi

GITHUB_CLIENT_ID=$(jq -r '.github.client_id' "$CONFIG_FILE")
GITHUB_CLIENT_SECRET=$(jq -r '.github.client_secret' "$CONFIG_FILE")

if [[ -z "$GITHUB_CLIENT_ID" ]] || [[ "$GITHUB_CLIENT_ID" == "null" ]]; then
    echo -e "${RED}✗${NC} GitHub client_id not found in config"
    exit 1
fi

if [[ -z "$GITHUB_CLIENT_SECRET" ]] || [[ "$GITHUB_CLIENT_SECRET" == "null" ]]; then
    echo -e "${RED}✗${NC} GitHub client_secret not found in config"
    exit 1
fi

echo -e "${GREEN}✓${NC} GitHub credentials loaded"

# Generate JWT signing secret
JWT_SECRET=$(openssl rand -base64 32)
echo -e "${GREEN}✓${NC} Generated JWT signing secret"

# Create IAM role for Lambda
echo -e "${BLUE}→${NC} Creating IAM role for Lambda..."
ROLE_NAME="GitHubOAuthLambdaRole"

TRUST_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
)

if aws iam get-role --role-name "$ROLE_NAME" &>/dev/null; then
    echo -e "${YELLOW}!${NC} Role $ROLE_NAME already exists"
else
    aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document "$TRUST_POLICY" \
        --description "Execution role for GitHub OAuth Lambda" \
        >/dev/null
    echo -e "${GREEN}✓${NC} Created IAM role: $ROLE_NAME"
fi

# Attach basic Lambda execution policy
aws iam attach-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-arn "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole" \
    2>/dev/null || true

# Create and attach STS AssumeRole policy
POLICY_NAME="GitHubOAuthAssumeRolePolicy"
CROSS_ACCOUNT_ROLE_ARN="arn:aws:iam::435415984226:role/SpawnDashboardCrossAccountReadRole"

STS_POLICY=$(cat <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "sts:AssumeRole",
      "Resource": "$CROSS_ACCOUNT_ROLE_ARN"
    }
  ]
}
EOF
)

if ! aws iam get-policy --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}" &>/dev/null; then
    aws iam create-policy \
        --policy-name "$POLICY_NAME" \
        --policy-document "$STS_POLICY" \
        --description "Allow GitHub OAuth Lambda to assume cross-account role" \
        >/dev/null
    echo -e "${GREEN}✓${NC} Created STS policy"
else
    echo -e "${YELLOW}!${NC} STS policy already exists"
fi

aws iam attach-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-arn "arn:aws:iam::${ACCOUNT_ID}:policy/${POLICY_NAME}" \
    2>/dev/null || true

# Wait for role to propagate
echo -e "${BLUE}→${NC} Waiting for IAM role to propagate..."
sleep 10

# Package Lambda function
echo -e "${BLUE}→${NC} Packaging Lambda function..."
cd "$LAMBDA_DIR"
zip -q -r /tmp/github-oauth-lambda.zip . -x "*.pyc" "__pycache__/*"
echo -e "${GREEN}✓${NC} Lambda package created"

# Create or update Lambda function
FUNCTION_NAME="github-oauth-bridge"
ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

echo -e "${BLUE}→${NC} Creating Lambda function..."

if aws lambda get-function --function-name "$FUNCTION_NAME" &>/dev/null; then
    # Update existing function
    aws lambda update-function-code \
        --function-name "$FUNCTION_NAME" \
        --zip-file fileb:///tmp/github-oauth-lambda.zip \
        >/dev/null

    aws lambda update-function-configuration \
        --function-name "$FUNCTION_NAME" \
        --environment "Variables={GITHUB_CLIENT_ID=$GITHUB_CLIENT_ID,GITHUB_CLIENT_SECRET=$GITHUB_CLIENT_SECRET,JWT_SECRET=$JWT_SECRET,REDIRECT_URI=https://api.spore.host/github/callback,FRONTEND_CALLBACK=https://spore.host/callback}" \
        --timeout 30 \
        --memory-size 256 \
        >/dev/null

    echo -e "${GREEN}✓${NC} Updated Lambda function: $FUNCTION_NAME"
else
    # Create new function
    aws lambda create-function \
        --function-name "$FUNCTION_NAME" \
        --runtime python3.11 \
        --role "$ROLE_ARN" \
        --handler handler.lambda_handler \
        --zip-file fileb:///tmp/github-oauth-lambda.zip \
        --timeout 30 \
        --memory-size 256 \
        --environment "Variables={GITHUB_CLIENT_ID=$GITHUB_CLIENT_ID,GITHUB_CLIENT_SECRET=$GITHUB_CLIENT_SECRET,JWT_SECRET=$JWT_SECRET,REDIRECT_URI=https://api.spore.host/github/callback,FRONTEND_CALLBACK=https://spore.host/callback}" \
        >/dev/null

    echo -e "${GREEN}✓${NC} Created Lambda function: $FUNCTION_NAME"
fi

# Get Lambda ARN
LAMBDA_ARN=$(aws lambda get-function --function-name "$FUNCTION_NAME" --query 'Configuration.FunctionArn' --output text)

# Create or get API Gateway
echo -e "${BLUE}→${NC} Setting up API Gateway..."
API_NAME="github-oauth-api"

# Check if API already exists
API_ID=$(aws apigatewayv2 get-apis --query "Items[?Name=='$API_NAME'].ApiId" --output text)

if [[ -z "$API_ID" ]]; then
    # Create new API
    API_ID=$(aws apigatewayv2 create-api \
        --name "$API_NAME" \
        --protocol-type HTTP \
        --query 'ApiId' \
        --output text)
    echo -e "${GREEN}✓${NC} Created API Gateway: $API_ID"
else
    echo -e "${YELLOW}!${NC} Using existing API Gateway: $API_ID"
fi

# Create Lambda integration
echo -e "${BLUE}→${NC} Creating API integration..."

INTEGRATION_ID=$(aws apigatewayv2 create-integration \
    --api-id "$API_ID" \
    --integration-type AWS_PROXY \
    --integration-uri "$LAMBDA_ARN" \
    --payload-format-version 2.0 \
    --query 'IntegrationId' \
    --output text 2>/dev/null || \
    aws apigatewayv2 get-integrations --api-id "$API_ID" --query 'Items[0].IntegrationId' --output text)

echo -e "${GREEN}✓${NC} Integration ID: $INTEGRATION_ID"

# Create route
echo -e "${BLUE}→${NC} Creating API route..."

ROUTE_KEY="GET /github/callback"
aws apigatewayv2 create-route \
    --api-id "$API_ID" \
    --route-key "$ROUTE_KEY" \
    --target "integrations/$INTEGRATION_ID" \
    >/dev/null 2>&1 || echo -e "${YELLOW}!${NC} Route already exists"

# Create or update stage
echo -e "${BLUE}→${NC} Creating API stage..."
STAGE_NAME='$default'

aws apigatewayv2 create-stage \
    --api-id "$API_ID" \
    --stage-name "$STAGE_NAME" \
    --auto-deploy \
    >/dev/null 2>&1 || echo -e "${YELLOW}!${NC} Stage already exists"

# Get API endpoint
API_ENDPOINT=$(aws apigatewayv2 get-api --api-id "$API_ID" --query 'ApiEndpoint' --output text)
CALLBACK_URL="${API_ENDPOINT}/github/callback"

echo -e "${GREEN}✓${NC} API Gateway configured"

# Grant API Gateway permission to invoke Lambda
echo -e "${BLUE}→${NC} Granting API Gateway invoke permission..."

aws lambda add-permission \
    --function-name "$FUNCTION_NAME" \
    --statement-id "apigateway-invoke" \
    --action "lambda:InvokeFunction" \
    --principal apigateway.amazonaws.com \
    --source-arn "arn:aws:execute-api:us-east-1:${ACCOUNT_ID}:${API_ID}/*/*/github/callback" \
    >/dev/null 2>&1 || echo -e "${YELLOW}!${NC} Permission already exists"

echo -e "${GREEN}✓${NC} Permissions configured"

# Clean up
rm /tmp/github-oauth-lambda.zip

echo ""
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║        Setup Complete! 🎉              ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Lambda Function:${NC} $FUNCTION_NAME"
echo -e "${BLUE}API Gateway ID:${NC} $API_ID"
echo -e "${BLUE}Callback URL:${NC} $CALLBACK_URL"
echo ""
echo -e "${YELLOW}⚠ IMPORTANT - Update GitHub OAuth App:${NC}"
echo ""
echo "1. Go to: https://github.com/settings/developers"
echo "2. Select your OAuth App: $GITHUB_CLIENT_ID"
echo "3. Update Authorization callback URL to:"
echo ""
echo -e "   ${GREEN}${CALLBACK_URL}${NC}"
echo ""
echo "4. Save changes"
echo ""
echo -e "${BLUE}Testing:${NC}"
echo "Visit https://spore.host and try GitHub login"
