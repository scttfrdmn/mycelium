## `lagotto poll`

Manually trigger a single poll of all active watches. This is for local testing; in production, polling runs on a Lambda schedule.

```
lagotto poll [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--daemon` |  | bool |  | Loop in the foreground, polling on --interval, until no active watches remain |
| `--interval` |  | duration | `5m0s` | Polling interval in --daemon mode (e.g. 30s, 5m) |
| `--mine` |  | bool |  | Only poll watches created by the calling identity |
| `--no-lease` |  | bool |  | Disable the per-watch processing lease (not recommended when multiple pollers run) |
| `--project` |  | string |  | Only poll watches with this project label (default: $LAGOTTO_PROJECT) |
| `--watch` |  | stringSlice |  | Only poll these watch IDs (comma-separated or repeated) |

