# Docker Hub Setup for Automated Builds

This guide explains how to configure Docker Hub credentials for automated spawn Docker image publishing via GitHub Actions.

## Overview

The GitHub Actions workflow `.github/workflows/docker-spawn.yml` automatically builds and publishes multi-architecture spawn Docker images on:
- Version tag pushes (e.g., `v0.13.2`)
- Manual workflow dispatch

**Images published:**
- `scttfrdmn/spawn:latest` (latest release)
- `scttfrdmn/spawn:v0.13.2` (version-specific tags)
- Multi-arch: `linux/amd64`, `linux/arm64`

## Prerequisites

- Docker Hub account
- Admin access to GitHub repository settings

## Setup Steps

### 1. Generate Docker Hub Access Token

1. Log into [Docker Hub](https://hub.docker.com)
2. Navigate to: **Account Settings** → **Security** → **Access Tokens**
3. Click **New Access Token**
4. Configure token:
   - **Description:** `GitHub Actions - mycelium`
   - **Access permissions:** Read, Write, Delete
5. Click **Generate**
6. **Copy the token immediately** (won't be shown again)

### 2. Add GitHub Repository Secrets

1. Navigate to: https://github.com/scttfrdmn/mycelium/settings/secrets/actions
2. Click **New repository secret**
3. Add first secret:
   - **Name:** `DOCKERHUB_USERNAME`
   - **Secret:** Your Docker Hub username (e.g., `scttfrdmn`)
4. Click **Add secret**
5. Add second secret:
   - **Name:** `DOCKERHUB_TOKEN`
   - **Secret:** Paste the access token from step 1
6. Click **Add secret**

### 3. Verify Setup

**Option A: Push a version tag**
```bash
git tag v0.13.3
git push origin v0.13.3
```

**Option B: Manual workflow dispatch**
1. Go to: https://github.com/scttfrdmn/mycelium/actions/workflows/docker-spawn.yml
2. Click **Run workflow**
3. Select branch and options
4. Click **Run workflow**

**Check workflow logs:**
- Navigate to Actions tab
- Click on the workflow run
- Verify "Log in to Docker Hub" step succeeds
- Verify images are pushed to Docker Hub

### 4. Verify Published Images

Check Docker Hub registry:
```bash
docker pull scttfrdmn/spawn:latest
docker pull scttfrdmn/spawn:v0.13.2
```

Or visit: https://hub.docker.com/r/scttfrdmn/spawn/tags

## Workflow Behavior

### Automatic Triggers

**Version tags** (e.g., `v0.13.2`):
- Builds multi-arch images (`linux/amd64`, `linux/arm64`)
- Pushes both versioned tag and `latest`
- Example: `v0.13.2` tag creates:
  - `scttfrdmn/spawn:0.13.2`
  - `scttfrdmn/spawn:latest`

**Main branch pushes** (with spawn changes):
- Builds single-arch image (`linux/amd64`)
- Does NOT push to Docker Hub
- Used for build verification only

### Manual Dispatch

Can manually trigger from Actions tab with options:
- **push_latest:** Push as `latest` tag (dev builds)

## Troubleshooting

### "Username and password required" Error

**Cause:** GitHub secrets not configured or named incorrectly

**Fix:**
1. Verify secrets exist at https://github.com/scttfrdmn/mycelium/settings/secrets/actions
2. Check exact names: `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` (case-sensitive)
3. Regenerate Docker Hub token if expired

### "unauthorized: incorrect username or password" Error

**Cause:** Invalid credentials or token expired

**Fix:**
1. Generate new Docker Hub access token
2. Update `DOCKERHUB_TOKEN` secret in GitHub
3. Retry workflow

### Build Succeeds but Images Not Visible

**Cause:** Private repository on Docker Hub

**Fix:**
1. Log into Docker Hub
2. Navigate to repository settings
3. Change visibility to Public

## Security Best Practices

**DO:**
- Use access tokens (not account password)
- Limit token permissions to Read/Write/Delete only
- Rotate tokens periodically (every 90 days)
- Use repository secrets (not environment secrets)

**DON'T:**
- Commit tokens to repository
- Share tokens between projects
- Use account password in CI/CD
- Store tokens in plaintext files

## Manual Build (Without GitHub Actions)

If credentials aren't configured, manually build and push:

```bash
cd spawn

# Log into Docker Hub
docker login -u scttfrdmn

# Build multi-arch image
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag scttfrdmn/spawn:v0.13.2 \
  --tag scttfrdmn/spawn:latest \
  --push \
  .
```

## Related Files

- `.github/workflows/docker-spawn.yml` - GitHub Actions workflow
- `spawn/Dockerfile` - Docker image definition
- `spawn/docs/how-to/docker.md` - Using spawn Docker image

## Support

Issues with Docker Hub setup:
- Docker Hub support: https://hub.docker.com/support/contact
- GitHub Actions docs: https://docs.github.com/en/actions/security-guides/encrypted-secrets
