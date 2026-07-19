---
description: "Connecting spore.host to Microsoft Teams gives you /spore commands in any channel and direct message notifications when your instances change state — useful…"
---

# Teams Setup

Connecting spore.host to Microsoft Teams gives you `/spore` commands in any channel and direct message notifications when your instances change state — useful whenever you're away from the terminal.

## What you'll get

**Slash commands** — type these in any Teams channel:
```
/spore list                    — all your registered instances
/spore status rstudio          — current state, type, URL, TTL countdown
/spore start rstudio           — start a stopped instance
/spore stop rstudio            — stop a running instance
/spore extend rstudio 4h       — extend the auto-terminate deadline
/spore hibernate rstudio       — hibernate (save RAM state, stop billing)
/spore connect                 — generate a one-time code for a collaborator
/spore notify rstudio          — subscribe to DM notifications for this instance
/spore help                    — command reference
```

**Proactive DM notifications** for lifecycle events:
- *rstudio terminates in 5 minutes*
- *bert-finetune has completed*
- *analysis has hibernated — idle timeout reached*
- *training received a Spot interruption notice*

## Prerequisites

- A Microsoft 365 tenant with Teams
- Azure Bot registration (free tier is sufficient)
- Self-hosted spore-bot Lambda deployed in your AWS account — see the [self-hosting guide](/guides/self-hosting) for full infrastructure setup

## Step 1: Create an Azure Bot

1. In the Azure Portal, go to **Azure Bot** and click **Create**
2. Name it `spore-bot` (or whatever you prefer)
3. Under **Configuration → Messaging endpoint**, enter your spore-bot Lambda URL:
   ```
   https://<your-lambda-url>/teams
   ```
4. Note your **Bot ID** (the app's Client ID) and **Client Secret** from the Azure Bot's **Configuration** page

## Step 2: Create the Teams app manifest

Create a file `manifest.json`:

```json
{
  "manifestVersion": "1.17",
  "version": "1.0.0",
  "id": "<YOUR_BOT_APP_ID>",
  "packageName": "host.spore.bot",
  "developer": {
    "name": "spore.host",
    "websiteUrl": "https://spore.host",
    "privacyUrl": "https://spore.host/privacy",
    "termsOfUseUrl": "https://spore.host/terms"
  },
  "name": { "short": "spore-bot" },
  "description": {
    "short": "Control your EC2 instances from Teams",
    "full": "Manage spore.host compute from Microsoft Teams"
  },
  "icons": { "color": "color.png", "outline": "outline.png" },
  "accentColor": "#3399FF",
  "bots": [
    {
      "botId": "<YOUR_BOT_APP_ID>",
      "scopes": ["personal", "team", "groupchat"],
      "commandLists": [
        {
          "scopes": ["personal", "team"],
          "commands": [
            { "title": "list", "description": "List your instances" },
            { "title": "status", "description": "Show instance status" },
            { "title": "help", "description": "Show command reference" }
          ]
        }
      ]
    }
  ],
  "composeExtensions": [
    {
      "botId": "<YOUR_BOT_APP_ID>",
      "commands": [
        {
          "id": "spore",
          "type": "action",
          "title": "spore",
          "description": "Control your spore.host instances"
        }
      ]
    }
  ],
  "permissions": ["identity", "messageTeamMembers"],
  "validDomains": ["spore.host"]
}
```

Replace `<YOUR_BOT_APP_ID>` with the Azure Bot's app ID (Client ID). Zip the manifest with your icon files and sideload it in Teams Admin Center, or publish it to your org's app catalog.

## Step 3: Connect your workspace

Register your Teams tenant and bot credentials with the spore-bot Lambda:

```sh
spawn notify workspace-add \
  --platform teams \
  --workspace-id <TEAMS_TENANT_ID> \
  --bot-token <AZURE_BOT_CLIENT_SECRET> \
  --signing-secret <AZURE_BOT_APP_ID>
```

Your tenant ID is visible in the Azure Portal under **Azure Active Directory → Overview** (the Directory/Tenant ID field).

## Step 4: Register an instance

Once the workspace is connected, register your instances so spore-bot can find them:

```sh
spawn notify register \
  --platform teams \
  --workspace-id <TEAMS_TENANT_ID> \
  --user-id <TEAMS_USER_ID> \
  --instance i-0a1b2c3d4e5f \
  --nickname rstudio
```

The `--nickname` is what you use in Teams commands (`/spore status rstudio`). To find your Teams user ID, run:

```sh
spawn notify list --platform teams --workspace-id <TEAMS_TENANT_ID>
```

Or use the `--user` flag with your email — spore-bot resolves it to a Teams user ID via the Graph API:

```sh
spawn notify register \
  --platform teams \
  --workspace-id <TEAMS_TENANT_ID> \
  --user you@university.edu \
  --instance i-0a1b2c3d4e5f \
  --nickname rstudio
```

Then enable the registration:

```sh
spawn notify enable \
  --platform teams \
  --user-id <TEAMS_USER_ID> \
  --workspace-id <TEAMS_TENANT_ID> \
  --nickname rstudio
```

## Step 5: Enable notifications at launch

Set `--slack-workspace` to your Teams tenant ID when launching to enable DM notifications:

```sh
spawn launch \
  --name rstudio \
  --instance-type r6i.2xlarge \
  --ttl 8h \
  --slack-workspace <TEAMS_TENANT_ID>
```

::: info
The `--slack-workspace` flag is used for both Slack and Teams. The platform is determined by the registered workspace record.
:::

Or save it as a default:

```sh
spawn defaults set slack-workspace <TEAMS_TENANT_ID>
```

## Adding collaborators

Give a collaborator access to an instance without giving them AWS credentials.

```sh
spawn notify register \
  --platform teams \
  --user collaborator@partner.edu \
  --workspace-id <TEAMS_TENANT_ID> \
  --instance i-0a1b2c3d4e5f \
  --nickname rstudio

spawn notify enable \
  --platform teams \
  --user collaborator@partner.edu \
  --workspace-id <TEAMS_TENANT_ID> \
  --nickname rstudio
```

Alternatively, generate a one-time connect code that the collaborator redeems themselves:

```
/spore connect     # generates a code like SPORE-XXXXXX
```

The collaborator runs:

```sh
spawn notify register --connect-code SPORE-XXXXXX --nickname rstudio
```

## Managing registrations

```sh
# List registrations for your workspace
spawn notify list --platform teams --workspace-id <TENANT_ID>

# Temporarily disable without deregistering
spawn notify disable \
  --platform teams \
  --user-id <TEAMS_USER_ID> \
  --workspace-id <TENANT_ID> \
  --nickname rstudio

# Permanently remove
spawn notify deregister \
  --platform teams \
  --user-id <TEAMS_USER_ID> \
  --workspace-id <TENANT_ID> \
  --nickname rstudio
```

## Troubleshooting

**Commands return "workspace not found"** — verify that `--workspace-id` matches the tenant ID used in `workspace-add`. Tenant IDs are UUIDs like `72f988bf-86f1-41af-91ab-2d7cd011db47`.

**Not receiving DM notifications** — ensure:
1. The instance was launched with `--slack-workspace <TENANT_ID>`
2. You've run `/spore notify <nickname>` in Teams to subscribe

**Teams bot not responding** — check the Lambda logs. The messaging endpoint in your Azure Bot configuration must match your Lambda URL exactly, including the `/teams` path suffix.

**Proactive messages blocked** — Teams requires the bot to have an active conversation reference before it can send proactive DMs. Send the bot any message (e.g., `hello`) to establish the conversation, then lifecycle notifications will flow.
