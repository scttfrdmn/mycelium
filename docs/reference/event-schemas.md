---
description: "The payload shape of spored's lifecycle events — for building webhooks and automation on top of spore.host."
---

# Event schemas

spored emits a lifecycle event when something significant happens to an instance
(TTL warning, completion, Spot interruption, …). Events are delivered as an HTTP
`POST` to the notification endpoint (`spawn:notify-url`), which routes them to
Slack/Teams. This page documents the payload shape so you can build automation on
top of it.

For *when* each event fires and its human-readable message, see
[Lifecycle Events](/reference/lifecycle-events). For enabling delivery to Slack or
a channel webhook, see [Lifecycle Notifications](/guides/notifications).

## Notification payload

Each event is a JSON object with these fields:

| Field | Type | Always present | Description |
|-------|------|----------------|-------------|
| `event_type` | string | yes | The event, e.g. `ttl_warning`, `completion`, `spot_interrupt`. See the [event type list](/reference/lifecycle-events#event-types). |
| `instance_name` | string | yes | The instance nickname (or ID if unnamed). |
| `instance_id` | string | yes | The EC2 instance ID. |
| `region` | string | yes | AWS region of the instance. |
| `platform` | string | yes | Delivery platform: `slack` or `teams`. |
| `workspace_id` | string | yes | Target Slack/Teams workspace for routing. |
| `command` | string | no | Slash-command app to route to (e.g. `/spore`) — disambiguates multiple registered apps. |
| `dns_name` | string | no | The instance's spore.host FQDN, when DNS registration is enabled. |
| `detail` | string | no | Free-text detail for the event (e.g. completion status/message). |
| `instance_identity_document` / `instance_identity_signature` / `pkcs7` | string | no | AWS instance identity attestation — proves the event genuinely originates from the named instance. |

```json
{
  "event_type": "completion",
  "instance_name": "bert-finetune",
  "instance_id": "i-0abc123def456xyz",
  "region": "us-east-1",
  "platform": "slack",
  "workspace_id": "T03ABCDEF",
  "command": "/spore",
  "dns_name": "bert-finetune.5k0zfnmq.spore.host",
  "detail": "1000 parameter combinations done"
}
```

::: tip Fire-and-forget delivery
Notifications are sent fire-and-forget: a slow or unavailable endpoint **never**
delays or cancels the underlying lifecycle action. Don't rely on a notification
being received as a precondition for a stop/terminate having happened.
:::

## Channel webhooks

Beyond the routed Slack/Teams DMs, you can post events to a shared Slack channel
webhook — set it with `spawn notify --webhook-url https://hooks.slack.com/...` (or
captured automatically during the Add-to-Slack OAuth flow). All events for any
instance in that workspace post to the channel. Today there is no per-event filter;
all events route to all subscribers. See
[Lifecycle Notifications](/guides/notifications).

## Verifying event authenticity

When present, the `instance_identity_document` + `instance_identity_signature`
(and `pkcs7`) fields carry AWS's signed EC2 instance identity, letting the receiver
confirm the event really came from the instance it names rather than a spoofed
caller. The hosted spore-bot Lambda verifies these before acting.

## Related

- **[Lifecycle Events](/reference/lifecycle-events)** — triggers, timing, messages
- **[Lifecycle Notifications](/guides/notifications)** — enabling delivery
- **[EC2 Tags](/reference/ec2-tags)** — the `spawn:notify-*` tags that configure delivery
