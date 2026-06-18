# lagotto command reference

## Global flags

All lagotto commands inherit these persistent flags.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `table` | Output format: `table`, `json` |
| `--verbose` | `-v` | bool | `false` | Enable verbose output |
| `--watches-table` | | string | `lagotto-watches` | DynamoDB table name for watches |
| `--history-table` | | string | `lagotto-match-history` | DynamoDB table name for match history |
| `--lang` | | string | (system) | Language for output: `en`, `es`, `fr`, `de`, `ja`, `pt` |
| `--no-emoji` | | bool | `false` | Disable emoji in output |
| `--accessibility` | | bool | `false` | Enable accessibility mode (implies `--no-emoji`) |

The `--watches-table` and `--history-table` flags let you point at non-default DynamoDB tables — useful when running multiple lagotto deployments in the same account.

---

## lagotto watch

Create a capacity watch for an instance type pattern.

```
lagotto watch <instance-type-pattern>
```

Registers a watch in DynamoDB. The deployed Lambda polls on a schedule and takes the configured action when capacity matching the pattern is found. If the polling schedule was self-disabled (no active watches), `lagotto watch` re-enables it.

**Pattern syntax:** Wildcards supported — `p5.*` matches all p5 sizes, `g5.xlarge` is an exact match. With `--service sagemaker` the pattern is an `ml.*` type (e.g. `ml.g5.2xlarge`), which is proxied to the correlated EC2 family (`g5.2xlarge`).

**Services (`--service`):**

| Service | Behavior |
|---------|----------|
| `ec2` (default) | Watch EC2 capacity directly. For `spawn`, the launch attempt is the capacity test. |
| `sagemaker` | Watch SageMaker `ml.*` capacity via the correlated EC2 family as a **proxy heuristic** (no real SageMaker capacity API exists). **Notify-only** — `spawn`/`hold` are rejected. Pattern must start with `ml.`. |

**Actions:**

| Action | Behavior |
|--------|----------|
| `notify` | Send a notification via the configured channels |
| `spawn` | Launch an instance using the provided spawn config file. A capacity failure keeps the watch `active` to retry; a terminal failure (bad AMI/IAM, exhausted quota) marks it `failed` |
| `hold` | Create a short-lived On-Demand Capacity Reservation to hold the matched capacity |

The watch **TTL is the only time limit** — there is no max-retry count. A
capacity failure (`InsufficientInstanceCapacity`) is not terminal: the watch
retries on each poll until it succeeds or the TTL expires.

**Examples:**
```sh
# Watch for any p5 instance and notify
lagotto watch "p5.*" --action notify --ttl 7d

# Watch for spot capacity under $10/hr
lagotto watch "p4d.24xlarge" --spot --max-price 10.00 --action notify

# Watch and auto-launch when capacity appears
lagotto watch "g5.xlarge" --action spawn --spawn-config my-job.yaml

# Watch specific regions only
lagotto watch "p5.48xlarge" --regions us-east-1,us-west-2

# Watch SageMaker ml.* capacity (proxy; notify-only)
lagotto watch "ml.g5.2xlarge" --service sagemaker --notify email:you@example.com
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--service` | | string | `ec2` | Capacity service: `ec2`, or `sagemaker` (EC2-proxy for `ml.*` types; notify-only) |
| `--regions` | `-r` | strings | (all enabled) | Regions to watch (comma-separated; empty = all enabled regions) |
| `--spot` | | bool | `false` | Watch for Spot capacity (default: On-Demand) |
| `--max-price` | | float | `0` | Maximum acceptable price per hour in USD; `0` = any price |
| `--action` | | string | `notify` | Action on match: `notify`, `spawn`, `hold` (SageMaker: `notify` only) |
| `--ttl` | | string | `24h` | How long to keep watching (e.g., `24h`, `7d`, `168h`) |
| `--notify` | | strings | | Notification channels: `email:user@example.com`, `webhook:https://...`, `sns:arn:...` |
| `--spawn-config` | | string | | Path to YAML file with spawn LaunchConfig (required when `--action spawn`) |
| `--project` | | string | (`$LAGOTTO_PROJECT`) | Project label for scoping a local `poll --daemon --project` in a shared account |

---

## lagotto launch

Schedule a future or recurring instance launch — by **time**, as opposed to `watch`, which fires on **capacity**.

```
lagotto launch (--at <time> | --after <delay> | --cron <expr>) --spawn-config <file>
```

Where `watch` waits for capacity to appear, `launch` fires at a clock time (`--at`), after a delay (`--after`), or on a recurring cron (`--cron`). The motivating case is launching into an [EC2 Capacity Block for ML](/tools/truffle#capacity-blocks-for-ml) at its reserved start time. The launched instance always carries a TTL.

Scheduled launches are driven by EventBridge Scheduler in the hosted poller stack, so they require **`lagotto deploy`** first (the per-launch schedule targets the poller Lambda in your account with a routing payload). A one-shot's schedule self-deletes after it fires; a cron schedule stays armed.

**Overlap policy (`--if-exists`):** when an instance with the same `Name` tag already exists at fire time:

| `--if-exists` | Behavior | Default for |
|---------------|----------|-------------|
| `skip` | Don't launch; treat the existing instance as the fulfillment | `--at` / `--after` (a Capacity Block must not double-book) |
| `launch` | Launch anyway — each fire is a fresh box | `--cron` |
| `replace` | Terminate the existing instance, then launch | — |

The dedup key is the instance `Name` tag (`--name`, or the spawn config's `name`).

**Examples:**
```sh
# Launch into a Capacity Block at its reserved start time (block.yaml sets
# reservation_id + capacity_block; --az matches the block's AZ)
lagotto launch --at 2026-07-01T08:00:00Z --az us-east-1a --spawn-config block.yaml

# Launch 6 hours from now
lagotto launch --after 6h --spawn-config job.yaml

# Recurring: every weekday at 09:00 UTC
lagotto launch --cron "0 9 ? * MON-FRI *" --spawn-config nightly.yaml
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--at` | string | | Fire once at this RFC3339 time (e.g. `2026-07-01T08:00:00Z`) |
| `--after` | string | | Fire once after this delay (e.g. `6h`, `30m`, `2d`) |
| `--cron` | string | | Fire on this cron schedule (e.g. `0 9 ? * MON-FRI *`) |
| `--spawn-config` | string | | YAML file with the spawn LaunchConfig (**required**) |
| `--if-exists` | string | `skip` (one-shot) / `launch` (cron) | Overlap policy: `skip`, `launch`, or `replace` |
| `--name` | string | (config's name) | Instance Name tag — the overlap dedup key |
| `--az` | string | | Availability zone (required to match a Capacity Block's AZ) |
| `--region` | string | (AWS config) | AWS region to launch in |
| `--stack-name` | string | `lagotto` | Deployed lagotto stack name (provides the poller target) |

---

## lagotto list

List your active watches.

```
lagotto list [--all]
```

By default shows only active watches. Use `--all` to include `matched`, `failed`, `expired`, and `cancelled` watches.

**Output columns:** Watch ID, Status, Pattern, Regions, Spot, Action, Expires

**Watch statuses:** `active` (being polled), `matched` (action succeeded), `failed` (terminal launch error — bad AMI/IAM, exhausted quota), `expired` (TTL elapsed), `cancelled`.

**Examples:**
```sh
lagotto list
lagotto list --all
lagotto list --output json
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | bool | `false` | Show all statuses (default: active only) |

---

## lagotto status

Show details of a specific watch.

```
lagotto status <watch-id>
```

**Output includes:** Watch ID, status, pattern, regions, Spot flag, max price, action, created/expires timestamps, last poll time, match count, and last match details (instance type, region, AZ, price, action taken, launched instance ID if applicable).

**Examples:**
```sh
lagotto status w-a1b2c3d4
lagotto status w-a1b2c3d4 --output json
```

---

## lagotto cancel

Cancel an active watch.

```
lagotto cancel <watch-id>
```

Marks the watch as cancelled in DynamoDB. The Lambda will stop polling for it on the next cycle. Cancelled watches cannot be re-activated; create a new watch instead.

**Examples:**
```sh
lagotto cancel w-a1b2c3d4
```

---

## lagotto extend

Extend a watch's TTL.

```
lagotto extend <watch-id> [--ttl <duration>]
```

Sets the watch expiry to `now + TTL`. If the watch has already expired, it is also reactivated (status set back to `active`) and the polling schedule is re-enabled.

**Examples:**
```sh
lagotto extend w-a1b2c3d4
lagotto extend w-a1b2c3d4 --ttl 48h
lagotto extend w-a1b2c3d4 --ttl 7d
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--ttl` | string | `24h` | New TTL from now (e.g., `24h`, `7d`) |

---

## lagotto history

Show the match history for your watches.

```
lagotto history [--watch-id <id>]
```

Without `--watch-id`, shows all matches across all your watches. With `--watch-id`, filters to a specific watch.

**Output columns:** Watch ID, Instance Type, Region, AZ, Price, Action Taken, Matched At

**Examples:**
```sh
lagotto history
lagotto history --watch-id w-a1b2c3d4
lagotto history --output json
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--watch-id` | string | | Filter to a specific watch |

---

## lagotto poll

Run one polling cycle, or loop in the foreground with `--daemon`.

```
lagotto poll [--daemon] [--interval <dur>] [scoping flags]
```

By default triggers a single poll of all active watches — the same logic the Lambda runs on its schedule. Useful for testing your watches without waiting for the next scheduled poll. With `--daemon` it loops on `--interval` (default 5m), so `lagotto watch --action spawn` works hands-off with **no** Lambda/EventBridge/CloudFormation — the infra-free alternative to the hosted poller.

Within a poll, a `spawn` watch actually **attempts a launch** (the real capacity test). A capacity failure leaves the watch `active` to retry next cycle; a terminal failure sets it `failed`. The poller is a self-terminating per-account singleton: when no active watches remain, the Lambda disables its own schedule (a new `lagotto watch` re-enables it).

**Scoping in a shared account:** by default a daemon services **every** active watch in the account. Scope it to your own so it doesn't drive (and launch) another project's watches:

```sh
lagotto poll --daemon --project fieldwork   # only that project's watches (or $LAGOTTO_PROJECT)
lagotto poll --daemon --mine                # only watches you created
lagotto poll --daemon --watch w-aaa,w-bbb   # only these watch IDs
```

A scoped daemon exits when **its** watches drain, not the whole account's. Before acting on a match, a poller claims a short **processing lease** on the watch, so two daemons — or a local daemon racing the hosted Lambda — can't both launch the same watch. A crashed poller's lease ages out automatically. Disable with `--no-lease` (not recommended when more than one poller runs).

**Examples:**
```sh
lagotto poll
lagotto poll --verbose
lagotto poll --daemon --interval 5m --project fieldwork
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--daemon` | bool | `false` | Loop in the foreground until no active watches in scope remain |
| `--interval` | duration | `5m` | Polling interval in `--daemon` mode (e.g. `30s`, `5m`) |
| `--project` | string | (`$LAGOTTO_PROJECT`) | Only poll watches with this project label |
| `--mine` | bool | `false` | Only poll watches created by the calling identity |
| `--watch` | strings | | Only poll these watch IDs (comma-separated or repeated) |
| `--no-lease` | bool | `false` | Disable the per-watch processing lease |

---

## lagotto version

Print the version number.

```
lagotto version
```

---

## lagotto completion

Generate shell completion scripts.

```
lagotto completion <shell>
```

**Supported shells:** `bash`, `zsh`, `fish`, `powershell`

**Setup examples:**
```sh
# bash
lagotto completion bash > /etc/bash_completion.d/lagotto

# zsh
lagotto completion zsh > "${fpath[1]}/_lagotto"
```
