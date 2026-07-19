---
description: "Spored emits lifecycle events when significant things happen to a running instance."
---

# Lifecycle Events

Spored emits lifecycle events when significant things happen to a running instance. Events are routed through the spore-bot Lambda to Slack DMs or Teams messages.

## Event types

| Event | When it fires | Default message |
|-------|--------------|-----------------|
| `ttl_warning` | 5 minutes before TTL expires | ⏱️ *name* terminates in 5 minutes |
| `ttl_expired` | At TTL expiry | 🔴 *name* has terminated — scheduled end time reached |
| `idle_warning` | 5 minutes before idle timeout | 💤 *name* will stop in 5 minutes — no activity detected |
| `idle_stopped` | At idle timeout (default) | ⏹️ *name* has stopped — idle timeout reached |
| `idle_hibernated` | At idle timeout with `--hibernate-on-idle` | 💤 *name* has hibernated — idle timeout reached |
| `completion` | When `spawn:completion-file` is detected | ✅ *name* has completed |
| `spot_interrupt` | On Spot interruption notice | ⚠️ *name* received a Spot interruption notice |
| `pre_stop_start` | When pre-stop hook begins | 🔄 *name* is running its shutdown task before terminating |

## Receiving events

Events reach you as Slack DMs or Teams messages. To enable:

1. Set `--slack-workspace` when launching (or `spawn defaults set slack-workspace <ID>`)
2. Run `/spore notify <nickname>` in Slack to subscribe your user

Without a subscription, events are posted to the channel webhook (if configured) but not sent as DMs.

## Warning thresholds

The 5-minute warning is fixed. If you need to respond (extend TTL, save work), you have that window. The warning fires once — subsequent checks within the same warning window don't re-trigger it.

## Completion events

The completion event fires when spored detects the file at `spawn:completion-file` (default: `/tmp/SPAWN_COMPLETE`). After the delay specified by `spawn:completion-delay` (default: `30s`), it executes the action in `spawn:on-complete`.

Typical pattern:

```sh
# At the end of your script
touch /tmp/SPAWN_COMPLETE
```

Spored sees the file, sends the `completion` notification, waits 30 seconds, then terminates the instance.
