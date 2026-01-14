# Dashboard Implementation Status

**Date:** 2025-12-30
**Status:** Infrastructure refactored, awaiting OAuth configuration

---

## Current Architecture

The dashboard has been **fully refactored** from server-side (Lambda + API Gateway) to **client-side architecture**.

### How It Works Now

```
User Browser
    ↓
1. Login via OIDC Provider (Globus Auth / Google / GitHub)
    ↓
2. Cognito Identity Pool exchanges OIDC token for temporary AWS credentials
    ↓
3. Browser uses AWS SDK for JavaScript to query EC2 directly
    ↓
4. Instances are filtered by spawn:iam-user tag (per-user isolation)
```

**Key Points:**
- No backend infrastructure needed (no Lambda, no API Gateway, no DynamoDB)
- Users query EC2 instances in **their own AWS accounts**
- Authentication via OIDC → Cognito Identity Pool → Temporary AWS credentials
- Natural multi-tenancy (each user sees only their instances)
- Per-user isolation via `spawn:iam-user` tag

---

## What's Been Completed

### ✅ Infrastructure Cleanup (Default Account: 752123829273)

All old server-side infrastructure has been **deleted**:
- Lambda function: `spawn-dashboard-api`
- API Gateway: `spawn-dashboard-api`
- DynamoDB table: `spawn-user-accounts`
- Lambda IAM role: `SpawnDashboardLambdaRole`

### ✅ Frontend Implementation

**Files Modified:**

1. **`web/js/auth.js`** (NEW)
   - Multi-provider OIDC authentication (Globus Auth, Google, GitHub)
   - Cognito Identity Pool integration
   - Credential caching in localStorage
   - OAuth 2.0 implicit flow implementation
   - Location: `/Users/scttfrdmn/src/mycelium/web/js/auth.js`

2. **`web/js/main.js`** (UPDATED)
   - Client-side EC2 queries using AWS SDK
   - Multi-region parallel queries (10 regions via Promise.allSettled)
   - Per-user filtering by `spawn:iam-user` tag
   - Dashboard UI rendering
   - Location: `/Users/scttfrdmn/src/mycelium/web/js/main.js`

3. **`web/index.html`** (UPDATED)
   - Added login section with OIDC provider buttons
   - Added AWS SDK script import
   - Added user display section
   - Location: `/Users/scttfrdmn/src/mycelium/web/index.html`

4. **`web/css/style.css`** (UPDATED)
   - Login UI styling
   - Dashboard styling
   - Badge variants for instance states
   - Location: `/Users/scttfrdmn/src/mycelium/web/css/style.css`

### ✅ Cognito Setup Script

**File:** `scripts/setup-dashboard-cognito.sh`
- Creates Cognito Identity Pool for OIDC authentication
- Creates IAM roles with EC2 read permissions for authenticated users
- Supports multiple OIDC providers (Globus Auth, Google, GitHub)
- Grants temporary credentials with permissions:
  - `ec2:DescribeInstances`
  - `ec2:DescribeRegions`
  - `ec2:DescribeTags`
  - `sts:GetCallerIdentity`
- Location: `/Users/scttfrdmn/src/mycelium/scripts/setup-dashboard-cognito.sh`

### ✅ Per-User Isolation in Spawn CLI

**File:** `spawn/pkg/aws/client.go`
- Added `spawn:iam-user` tag to all EC2 instances
- Tag contains IAM user ARN from `STS GetCallerIdentity`
- Enables filtering instances by user (not just account)
- Location: `/Users/scttfrdmn/src/mycelium/spawn/pkg/aws/client.go:775-789` (GetCallerIdentityInfo)
- Location: `/Users/scttfrdmn/src/mycelium/spawn/pkg/aws/client.go:410-420` (buildTags)

### ✅ Deployment

- Website deployed to S3: `s3://spore-host-website/`
- CloudFront cache invalidated (Distribution: E50GL663TTL0I)
- Accessible at: https://spore.host

---

## What's Pending (ON HOLD)

The following steps are **required** before users can authenticate and use the dashboard:

### 1. Register OAuth Applications

Register OAuth apps with each provider using redirect URI: `https://spore.host/callback`

**Globus Auth:**
- URL: https://developers.globus.org/
- Application type: Web application
- Redirect URI: `https://spore.host/callback`
- Scopes: `openid profile email`
- Save: **Client ID**

**Google:**
- URL: https://console.cloud.google.com/apis/credentials
- Create: OAuth 2.0 Client ID (Web application)
- Authorized redirect URIs: `https://spore.host/callback`
- Save: **Client ID**

**GitHub:**
- URL: https://github.com/settings/developers
- Create: New OAuth App
- Homepage URL: `https://spore.host`
- Authorization callback URL: `https://spore.host/callback`
- Save: **Client ID**

### 2. Run Cognito Setup Script

After obtaining OAuth Client IDs, run:

```bash
cd /Users/scttfrdmn/src/mycelium/scripts
AWS_PROFILE=default ./setup-dashboard-cognito.sh
```

The script will:
- Prompt for Client IDs from each provider
- Create Cognito Identity Pool
- Create IAM roles for authenticated/unauthenticated users
- Grant EC2 read permissions
- Output **Identity Pool ID** (save this)

### 3. Update Frontend Configuration

Edit `/Users/scttfrdmn/src/mycelium/web/js/auth.js`:

```javascript
const AUTH_CONFIG = {
    region: 'us-east-1',
    identityPoolId: 'us-east-1:XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX', // TODO: Set from Cognito setup

    providers: {
        globus: {
            name: 'Globus Auth',
            clientId: 'XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX', // TODO: Set from Globus
            authEndpoint: 'https://auth.globus.org/v2/oauth2/authorize',
            scope: 'openid profile email',
            cognitoKey: 'auth.globus.org'
        },
        google: {
            name: 'Google',
            clientId: 'XXXXXXXXXXXX-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX.apps.googleusercontent.com', // TODO: Set from Google
            authEndpoint: 'https://accounts.google.com/o/oauth2/v2/auth',
            scope: 'openid profile email',
            cognitoKey: 'accounts.google.com'
        },
        github: {
            name: 'GitHub',
            clientId: 'XXXXXXXXXXXXXXXXXXXX', // TODO: Set from GitHub
            authEndpoint: 'https://github.com/login/oauth/authorize',
            scope: 'read:user user:email',
            cognitoKey: null // GitHub requires custom handling
        }
    }
};
```

### 4. Redeploy Website

After configuration:

```bash
cd /Users/scttfrdmn/src/mycelium/web
AWS_PROFILE=default ./deploy.sh
```

### 5. Test End-to-End

Launch a test instance and verify dashboard shows it:

```bash
# Launch instance (will automatically get spawn:iam-user tag)
AWS_PROFILE=aws ./bin/spawn launch --instance-type t3.micro --name test-dashboard --ttl 2h

# Login to https://spore.host
# Should see the test instance in dashboard
```

---

## Important Notes

### Account Configuration

**Default Account (752123829273):**
- Hosts website: S3 + CloudFront
- Runs Cognito Identity Pool
- Grants temporary EC2 read credentials to authenticated users
- **Does NOT query any EC2 instances**

**User's AWS Accounts:**
- Users authenticate with OIDC provider
- Get temporary AWS credentials from Cognito
- Browser queries EC2 instances **in their own account**
- Filtered by `spawn:iam-user` tag (their IAM user ARN)

### Security Model

- **Per-User Isolation:** Instances tagged with `spawn:iam-user` = IAM user ARN
- **Client-Side Queries:** Browser directly queries EC2 API (no shared backend)
- **Temporary Credentials:** Cognito issues short-lived AWS credentials (1 hour TTL)
- **Natural Multi-Tenancy:** Each user queries their own AWS account

### Files Reference

**Frontend:**
- `/Users/scttfrdmn/src/mycelium/web/index.html` - Dashboard UI
- `/Users/scttfrdmn/src/mycelium/web/js/auth.js` - Authentication logic
- `/Users/scttfrdmn/src/mycelium/web/js/main.js` - Dashboard API (client-side EC2 queries)
- `/Users/scttfrdmn/src/mycelium/web/css/style.css` - Styling

**Scripts:**
- `/Users/scttfrdmn/src/mycelium/scripts/setup-dashboard-cognito.sh` - Cognito setup
- `/Users/scttfrdmn/src/mycelium/web/deploy.sh` - Website deployment

**Spawn CLI:**
- `/Users/scttfrdmn/src/mycelium/spawn/pkg/aws/client.go` - Per-user tagging implementation

---

## Testing Without OAuth Setup

You can test the dashboard infrastructure without completing OAuth setup by using AWS IAM credentials directly:

```javascript
// In browser console at https://spore.host
AWS.config.update({
    region: 'us-east-1',
    credentials: new AWS.Credentials({
        accessKeyId: 'AKIA...',
        secretAccessKey: '...',
        sessionToken: '...' // if using STS
    })
});

// Then call dashboard functions
await DashboardAPI.listInstances();
```

However, production usage requires completing the OAuth setup.

---

## Next: Account Refactoring

**Current Task:** Refactor AWS account configuration before completing OAuth setup.

**When ready to resume dashboard work:** Refer to "What's Pending" section above.
