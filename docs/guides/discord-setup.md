# Discord Setup

Connecting spore.host to Discord posts instance lifecycle notifications —
TTL warnings, idle stops, spot interruptions, completion — to a channel of your
choice, as color-coded embeds.

> **Status:** Phase 1 ships **notifications** (this guide). Discord slash
> commands (`/spore list|status|extend|stop`) are Phase 2 — see
> [spore-host/spawn#2](https://github.com/spore-host/spawn/issues/2).

## What you'll get

Channel notifications for lifecycle events, as Discord embeds color-coded by
severity:
- ⏱️ **rstudio terminates in 5 minutes** (yellow)
- ✅ **bert-finetune has completed** (green)
- 💤 **analysis has hibernated — idle timeout reached** (red)
- ⚠️ **training received a Spot interruption notice** (yellow)

Each embed carries the instance ID, region, and URL.

## Step 1: Create a Discord channel webhook

The simplest delivery path — no bot user required for notifications.

1. In Discord, open the target channel's **Edit Channel → Integrations →
   Webhooks**.
2. **New Webhook**, name it `spore-bot`, pick the channel, **Copy Webhook URL**
   (looks like `https://discord.com/api/webhooks/<id>/<token>`).

## Step 2 (optional, for Phase 2): Create a Discord application

Only needed later for slash commands; skip if you just want notifications.

1. Go to [discord.com/developers/applications](https://discord.com/developers/applications)
   → **New Application**, name it `spore-bot`.
2. On the **General Information** page, copy the **Public Key** (hex) — this is
   what verifies inbound interactions (Discord has no signing secret).
3. Under **Bot**, add a bot and copy its **token** if you want bot DMs later.

## Step 3: Register the workspace with spawn

Store the webhook (and, optionally, the application public key for Phase 2) so
the spore-bot service can deliver to your channel. Your Discord **server (guild)
ID** is the workspace ID — enable Developer Mode in Discord, right-click the
server, **Copy Server ID**.

```bash
spawn notify workspace-add \
  --platform discord \
  --workspace-id <GUILD_ID> \
  --workspace-name "My Server" \
  --webhook-url "https://discord.com/api/webhooks/<id>/<token>" \
  --public-key <APPLICATION_PUBLIC_KEY>     # optional; for Phase 2 slash commands
```

`--public-key` is required only when you intend to use slash commands; for
notifications alone, just the `--webhook-url` is needed. (Discord uses the
public key for Ed25519 interaction verification, not a signing secret — so
`--signing-secret` does not apply to `--platform discord`.)

## Step 4: Launch with Discord notifications

Tell the instance to route lifecycle notifications to Discord:

```bash
spawn launch rstudio --instance-type r7i.xlarge \
  --slack-workspace <GUILD_ID> \
  --notify-platform discord
```

(`--slack-workspace` is the workspace-id flag — it carries the Discord guild ID
here; the flag name is shared across platforms.) The instance is tagged
`spawn:notify-platform=discord` and `spawn:slack-workspace-id=<GUILD_ID>`, and
spored sends its lifecycle events to the spore-bot `/notify` endpoint, which
posts the embed to your channel webhook.

That's it — you'll see embeds in the channel as the instance warns, stops, or
completes.
