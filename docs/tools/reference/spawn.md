# spawn command reference

## Global flags

All spawn commands inherit these persistent flags.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `table` | Output format: `table`, `json` |
| `--verbose` | `-v` | bool | `false` | Enable verbose output |
| `--no-color` | | bool | `false` | Disable colorized output |
| `--lang` | | string | (system) | Language for output: `en`, `es`, `fr`, `de`, `ja`, `pt` |
| `--no-emoji` | | bool | `false` | Disable emoji in output |
| `--accessibility` | | bool | `false` | Enable accessibility mode (implies `--no-emoji`) |

---

## spawn launch

Launch an EC2 instance with automatic lifecycle management.

```
spawn launch <name> [flags]
```

`<name>` is required. It sets the EC2 `Name` tag, the DNS hostname (`<name>.<account>.spore.host`), and the instance's own hostname. Names must be unique within your account.

**Examples:**
```sh
# Basic launch — 1h idle timeout auto-applied if no --ttl
spawn launch my-job --instance-type c6a.4xlarge

# GPU training with TTL, Spot, and auto-terminate on completion
spawn launch train --instance-type p4d.24xlarge --spot --ttl 48h --on-complete terminate

# Job array (3 identical instances)
spawn launch workers --instance-type m7i.xlarge --count 3 --job-array-name workers

# Parameter sweep from YAML file
spawn launch sweep --instance-type c6i.2xlarge --param-file params.yaml --cartesian

# MPI cluster with EFA
spawn launch mpi-job --instance-type hpc7i.48xlarge --count 4 --mpi --efa

# With pre-stop S3 sync before lifecycle termination
spawn launch analysis --instance-type r7i.4xlarge --ttl 8h \
  --pre-stop "aws s3 sync /results s3://bucket/run-001"

# Larger root volume
spawn launch big-disk --instance-type m7i.large --volume-size 200
```

**JSON output:** With `-o json`, the TUI progress display is suppressed and
`spawn launch` emits a parseable JSON **array** of launched instances, each with
`instance_id`, `name`, `instance_type`, `region`, `public_ip`, `state`, and
`dns`. Capture the ID in scripts with:

```sh
id=$(spawn launch my-job --instance-type c6a.large -y -o json | jq -r '.[0].instance_id')
```

### Instance configuration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--instance-type` | string | | EC2 instance type |
| `--region` | string | (config default) | AWS region |
| `--az` | string | | Specific availability zone |
| `--ami` | string | (auto-detects AL2023) | AMI ID. Use `auto` (or omit) to auto-detect the latest AL2023 AMI |
| `--volume-size` | int | (AMI default) | Root EBS volume size in GiB. Use for AMI-baking or large datasets that exceed the default root volume |
| `--vpc` | string | | VPC ID (auto-creates if unset) |
| `--subnet` | string | | Subnet ID |
| `--security-group` | string | | Security group ID |
| `--key-pair` | string | | SSH key pair name |

### Capacity

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--spot` | bool | `false` | Launch as Spot instance |
| `--spot-max-price` | string | | Maximum Spot price (e.g., `0.50`) |
| `--use-reservation` | bool | `false` | Use On-Demand Capacity Reservation |
| `--reservation-id` | string | | Specific reservation ID to use |

### Lifecycle

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--ttl` | string | | Auto-terminate after duration (e.g., `8h`, `2d`). If neither `--ttl` nor `--idle-timeout` are set, a 1h idle timeout is applied automatically. |
| `--idle-timeout` | string | `1h` | Auto-terminate if no activity for this duration. Disabled if `--ttl` is set unless also explicitly specified. |
| `--no-timeout` | bool | `false` | Disable all automatic timeout (creates zombie risk — use with caution) |
| `--hibernate` | bool | `false` | Enable hibernation support |
| `--hibernate-on-idle` | bool | `false` | Hibernate instead of terminate when idle |
| `--on-complete` | string | | Action when workload signals completion: `terminate`, `stop`, `hibernate` |
| `--completion-file` | string | `/tmp/SPAWN_COMPLETE` | File path that signals completion |
| `--completion-delay` | string | `30s` | Grace period after completion signal |
| `--pre-stop` | string | | Shell command to run before any lifecycle-triggered stop/terminate |
| `--pre-stop-timeout` | string | `5m` (Spot: `90s`) | Max time for `--pre-stop` command |
| `--session-timeout` | string | `30m` | Auto-logout idle SSH sessions (`0` to disable) |

### Notifications and idle detection

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--slack-workspace` | string | | Slack workspace ID for lifecycle notifications (e.g., `T03NE3GTY`) |
| `--active-ports` | string | | Comma-separated TCP ports to monitor — won't idle-terminate while any are active (e.g., `8787` for RStudio, `8787,8888` for RStudio+Jupyter) |
| `--active-processes` | string | | Comma-separated process names to monitor — won't idle-terminate while any are running (e.g., `rsession`, `rsession,jupyter`) |

### Networking

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--dns` | string | | Override DNS name if different from `--name` |
| `--dns-domain` | string | | Custom DNS domain |
| `--dns-api-endpoint` | string | | Custom DNS API endpoint |

### Startup

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--user-data` | string | | Inline user-data script or `@/path/to/file` |
| `--user-data-file` | string | | Path to user-data script file |
| `--plugin` | strings | | Plugin to install at launch (repeatable, e.g., `--plugin my-plugin@1.0`) |
| `--config` | string | | Launch config YAML file (supports `plugins:` list) |
| `--command` | string | | Command to run on all instances after setup |

### Shared storage

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--efs-id` | string | | EFS filesystem ID to mount (e.g., `fs-abc123`) |
| `--efs-mount-point` | string | `/efs` | EFS mount point |
| `--efs-profile` | string | `general` | EFS performance profile: `general`, `max-io`, `max-throughput`, `burst` |
| `--efs-mount-options` | string | | Custom EFS mount options (overrides profile) |
| `--fsx-id` | string | | Existing FSx Lustre filesystem ID to mount |
| `--fsx-create` | bool | `false` | Create new FSx Lustre (`PERSISTENT_2`, Lustre 2.15, compatible with AL2023) with S3 backing via Data Repository Association. Writes `spawn:fsx-id`, `spawn:fsx-mount-name`, `spawn:fsx-mount-point` tags on all instances. |
| `--fsx-id` | string | | Existing FSx filesystem ID. Writes `spawn:fsx-id`, `spawn:fsx-mount-name`, `spawn:fsx-mount-point` tags so boot scripts can auto-mount without hardcoding the ID. |
| `--fsx-recall` | string | | Recall FSx by CloudFormation stack name |
| `--fsx-storage-capacity` | int | `1200` | FSx storage in GB (1200, 2400, or multiples of 2400) |
| `--fsx-throughput` | int | `125` | PERSISTENT_2 throughput in MB/s/TiB — valid values: `125`, `250`, `500`, `1000` |
| `--fsx-s3-bucket` | string | | S3 bucket for FSx (required with `--fsx-create`) |
| `--fsx-import-path` | string | | S3 import path |
| `--fsx-export-path` | string | | S3 export path |
| `--fsx-mount-point` | string | `/fsx` | FSx mount point |

### Job arrays

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--count` | int | `1` | Number of instances to launch |
| `--job-array-name` | string | | Job array name (required when `--count > 1`) |
| `--instance-names` | string | | Instance name template (e.g., `worker-{index}`) |

### MPI clusters

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--mpi` | bool | `false` | Enable MPI cluster setup (requires `--count > 1`) |
| `--efa` | bool | `false` | Enable Elastic Fabric Adapter for ultra-low-latency MPI |
| `--mpi-processes-per-node` | int | (vCPU count) | MPI processes per node |
| `--mpi-command` | string | | Command to run via `mpirun` |
| `--skip-mpi-install` | bool | `false` | Skip MPI installation (for AMIs with MPI pre-installed) |
| `--placement-group` | string | | Use an existing AWS Placement Group by name |
| `--auto-placement-group` | bool | `true` | Auto-create a cluster placement group named `spawn-mpi-<job-array-name>` and wait for it to become available before launching instances |

### Parameter sweeps

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--param-file` | string | | Path to parameter file (JSON, YAML, or CSV) |
| `--params` | string | | Inline JSON parameters |
| `--cartesian` | bool | `false` | Generate cartesian product of parameter lists |
| `--sweep-name` | string | (auto) | Human-readable sweep identifier |
| `--max-concurrent` | int | `0` | Max simultaneous instances (`0` = unlimited) |
| `--max-concurrent-per-region` | int | `0` | Max simultaneous instances per region |
| `--launch-delay` | string | `0s` | Delay between instance launches |
| `--detach` | bool | `false` | Run sweep orchestration in Lambda (auto-enabled for sweeps) |
| `--no-detach` | bool | `false` | Disable auto-detach (requires `--ttl` or `--idle-timeout`) |
| `--budget` | float | `0` | Budget limit in dollars (`0` = no limit) |
| `--cost-limit` | float | `0` | Terminate when spend reaches this amount (`0` = disabled) |
| `--estimate-only` | bool | `false` | Show cost estimate and exit without launching |
| `--yes` / `-y` | bool | `false` | Auto-approve cost estimate |
| `--mode` | string | `balanced` | Distribution mode: `balanced` or `opportunistic` |

### Region constraints (for sweeps)

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--regions-include` | strings | | Only use these regions (wildcards: `us-*`) |
| `--regions-exclude` | strings | | Exclude these regions (wildcards: `eu-*`) |
| `--regions-geographic` | strings | | Geographic filter: `us`, `eu`, `ap`, `north-america` |
| `--proximity-from` | string | | Prefer regions close to this region |
| `--cost-tier` | string | | Prefer cost tier: `low`, `standard`, `premium` |

### IAM

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--iam-role` | string | | IAM role name (creates if doesn't exist) |
| `--iam-policy` | strings | | Service-level policies (e.g., `s3:ReadOnly`) |
| `--iam-managed-policies` | strings | | AWS managed policy ARNs |
| `--iam-policy-file` | string | | Custom IAM policy JSON file |
| `--iam-trust-services` | strings | `ec2` | Services that can assume the role |

### Compliance

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--nist-800-171` | bool | `false` | Enable NIST 800-171 Rev 3 compliance mode |
| `--nist-800-53` | string | | Enable NIST 800-53 compliance level: `low`, `moderate`, `high` |
| `--compliance-strict` | bool | `false` | Fail on compliance warnings (default: warn only) |

### Workflow integration

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--output-id` | string | | Write instance/sweep ID to file for scripting |
| `--wait` | bool | `false` | Wait for sweep/launch to complete |
| `--wait-timeout` | string | `0` | Timeout for `--wait` (`0` = no timeout) |
| `--team` | string | | Team ID for team-shared instance access |
| `--interactive` | bool | `false` | Force interactive wizard |
| `--quiet` | bool | `false` | Minimal output |

---

## spawn list

List running instances.

```
spawn list [flags]
spawn ls [flags]
```

Groups output by parameter sweeps, job arrays, and standalone instances. Shows columns: Instance ID, Name, Type, State, IAM Role, AZ, Age, TTL, Public IP, Spot.

**Examples:**
```sh
spawn list
spawn ls --regions us-east-1,us-west-2
spawn list --state running --instance-family m7i
spawn list --sweep-name my-sweep
spawn list --output json
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--region` | | string | (all) | Filter by a single region |
| `--regions` | `-r` | strings | (all) | Filter by multiple regions (comma-separated) |
| `--az` | | string | | Filter by availability zone |
| `--state` | | string | | Filter by state: `running`, `stopped`, `pending`, etc. |
| `--instance-type` | | string | | Filter by exact instance type |
| `--instance-family` | | string | | Filter by instance family (e.g., `m7i`) |
| `--tag` | | strings | | Filter by tag `key=value` (repeatable) |
| `--job-array-id` | | string | | Filter by job array ID |
| `--job-array-name` | | string | | Filter by job array name |
| `--sweep-id` | | string | | Filter by sweep ID |
| `--sweep-name` | | string | | Filter by sweep name |

---

## spawn status

Show status of an instance or parameter sweep.

```
spawn status <instance-id-or-name>
```

Connects to the instance via SSH and runs `spored status`, reporting lifecycle state, TTL countdown, idle timer, and completion signal.

**Examples:**
```sh
spawn status my-job
spawn status i-0abc123
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--check-complete` | bool | `false` | Exit with standardized codes: `0`=complete, `1`=failed, `2`=running, `3`=error |

`--check-complete` inspects the instance's completion file (`spawn:completion-file`,
default `/tmp/SPAWN_COMPLETE`): `0` if present, `1` if it reports a failure status,
`2` if absent (still running), `3` on error. Works for a single instance (v0.36.6+)
and, via `spawn sweep status --check-complete`, for parameter sweeps.

---

## spawn connect

Open an SSH session to an instance, or run a single command and return.

```
spawn connect <instance-id-or-name> [-- <command>...]
spawn ssh <instance-id-or-name>
```

Resolves the instance by ID or name, finds the SSH key automatically from `~/.ssh/`, and invokes `ssh`. Falls back to AWS Session Manager if no public IP is available or `--session-manager` is set. For DCV app streaming instances, opens the browser session instead.

Pass `-- <command>` to run a command non-interactively (one-shot mode). The command is wrapped in `bash -c '...'` on the remote side, so compound operators (`&&`, `;`, `&`, pipes) and background jobs work correctly. Single quotes in the command are escaped automatically.

If the instance is `stopped` or hibernated, spawn automatically starts it and waits for it to reach `running` before connecting. Use `--no-start` to disable this behaviour.

When multiple instances share the same name, connect prefers the `running` instance. If multiple running instances share the name, the command fails with a list of the running instances so you can specify an ID.

**Examples:**
```sh
spawn connect my-job                                               # interactive shell
spawn ssh i-0abc123
spawn connect my-job --user ubuntu --port 2222
spawn connect my-job --session-manager

# One-shot mode — single command
spawn connect my-job -- 'tail -20 /tmp/run.log'
spawn connect my-job -- 'aws s3 cp s3://bucket/run.sh /tmp/run.sh'

# Compound commands work — remote shell interprets && ; & etc.
spawn connect my-job -- 'cmd1 && cmd2'
spawn connect my-job -- 'nohup bash /tmp/run.sh > /tmp/run.log 2>&1 &'
spawn connect my-job -- 'aws s3 cp s3://bucket/run.sh /tmp/ && bash /tmp/run.sh'
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--user` | string | `ec2-user` | SSH username |
| `--key` | string | (auto-detect) | SSH private key path |
| `--port` | int | `22` | SSH port |
| `--session-manager` | bool | `false` | Use AWS Session Manager instead of SSH |
| `--no-start` | bool | `false` | Do not automatically start a stopped/hibernated instance |

---

## spawn extend

Extend an instance's TTL.

```
spawn extend <instance-id-or-name> <duration>
spawn extend <duration> --job-array-id <id>
spawn extend <duration> --job-array-name <name>
```

TTL extension is anchored to the original launch time — extending adds to the current deadline, not to the current wall clock. After updating the tag, spored is signaled to reload its configuration.

**Examples:**
```sh
spawn extend my-job 4h
spawn extend i-0abc123 1d
spawn extend 2h --job-array-name workers
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--job-array-id` | string | | Extend all instances in job array by ID |
| `--job-array-name` | string | | Extend all instances in job array by name |

---

## spawn stop / start / hibernate

Stop, start, or hibernate an instance.

```
spawn stop <instance-id-or-name>
spawn start <instance-id-or-name>
spawn hibernate <instance-id-or-name>
```

All three commands accept `--job-array-id` or `--job-array-name` to operate on an entire job array at once. `stop` and `hibernate` preserve EBS volumes — to permanently destroy an instance, use `spawn terminate`.

---

## spawn terminate

Permanently terminate an instance. Unlike `stop`/`hibernate` (which preserve EBS
volumes), `terminate` destroys the instance and its EBS volumes. This is
irreversible, so it prompts for confirmation by default.

```
spawn terminate <instance-id-or-name>
spawn terminate --job-array-id <id>
spawn terminate --job-array-name <name>
```

**Examples:**
```sh
spawn terminate my-instance         # prompts, then terminates (EBS destroyed)
spawn terminate my-instance -y      # skip the confirmation prompt
spawn terminate --job-array-name workers   # terminate the whole array
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--yes` | `-y` | bool | `false` | Skip the confirmation prompt |
| `--job-array-id` | | string | | Terminate all instances in job array by ID |
| `--job-array-name` | | string | | Terminate all instances in job array by name |

---

## spawn cancel

Cancel a parameter sweep and terminate all associated instances.

```
spawn cancel --sweep-id <sweep-id>
```

---

## spawn sweep

Manage parameter sweeps after they have been launched.

```
spawn sweep <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `list` | `spawn sweep list` | List parameter sweeps |
| `status` | `spawn sweep status <sweep-id>` | Show sweep progress and instance breakdown |
| `cancel` | `spawn sweep cancel <sweep-id>` | Cancel and terminate all sweep instances |
| `resume` | `spawn sweep resume <sweep-id>` | Resume an interrupted sweep from checkpoint |
| `collect` | `spawn sweep collect <sweep-id>` | Download and aggregate results |

**`spawn sweep list` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--status` | string | (all) | Filter by status: `RUNNING`, `COMPLETED`, `FAILED`, `CANCELLED` |
| `--last` | int | `20` | Show last N sweeps |
| `--since` | string | | Show sweeps created after date (`YYYY-MM-DD`) |
| `--region` | string | | Filter by region |

**`spawn sweep status` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--check-complete` | bool | `false` | Exit with standardized codes: `0`=complete, `1`=failed, `2`=running, `3`=error |

**`spawn sweep resume` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max-concurrent` | int | `0` | Override max concurrent instances (`0` = use original) |
| `--detach` | bool | `false` | Run orchestration in Lambda |

**`spawn sweep collect` flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output-file` | `-f` | string | `results.json` | Output file path |
| `--format` | | string | `json` | Output format: `json`, `csv`, `jsonl` |
| `--s3-prefix` | | string | | Custom S3 prefix |
| `--metric` | | string | | Metric field for ranking |
| `--best` | | int | `0` | Show top N results (`0` = all) |
| `--regions` | | string | | Comma-separated regions to collect from |

---

## spawn collect-results

Collect and aggregate results from a parameter sweep.

```
spawn collect-results --sweep-id <sweep-id>
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--sweep-id` | | string | (required) | Sweep ID to collect from |
| `--output` | `-o` | string | `results.json` | Output file path |
| `--format` | | string | `json` | Output format: `json`, `csv`, `jsonl` |
| `--s3-prefix` | | string | | Custom S3 prefix for results |
| `--metric` | | string | | Metric field name for ranking results |
| `--best` | | int | | Show only top N results |
| `--regions` | | string | | Comma-separated regions to collect from |

---

## spawn schedule

Manage scheduled parameter sweep executions.

```
spawn schedule <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `create` | `spawn schedule create <params-file>` | Schedule a sweep for future execution |
| `list` | `spawn schedule list` | List all schedules |
| `cancel` | `spawn schedule cancel <schedule-id>` | Cancel a schedule |
| `update` | `spawn schedule update <schedule-id>` | Update schedule timing |

**`spawn schedule create` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--at` | string | | One-time execution time (ISO 8601, e.g., `2026-06-01T14:00:00Z`) |
| `--cron` | string | | Recurring cron expression (e.g., `0 9 * * MON`) |
| `--name` | string | | Schedule name |
| `--max-executions` | int | | Maximum number of executions |
| `--end-after` | string | | Stop after duration (e.g., `30d`) |
| `--timezone` | string | | Timezone for cron (e.g., `America/New_York`) |

---

## spawn queue

Manage batch job queues for sequential workload execution.

```
spawn queue <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `status` | `spawn queue status <instance-id>` | Show queue execution status |
| `results` | `spawn queue results <queue-id>` | Download results from S3 |
| `template list` | `spawn queue template list` | List available queue templates |
| `template show` | `spawn queue template show <name>` | Show template details |
| `template generate` | `spawn queue template generate` | Generate config from template |

**`spawn queue results` flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `.` | Output directory for results |

---

## spawn autoscale

Manage auto-scaling job arrays that grow and shrink in response to queue depth or custom metrics.

```
spawn autoscale <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `launch` | `spawn autoscale launch` | Launch an auto-scaling group |
| `update` | `spawn autoscale update <name>` | Update desired/min/max capacity |
| `status` | `spawn autoscale status [name]` | Show group status (omit name for all) |
| `health` | `spawn autoscale health <name>` | Show instance health details |
| `pause` | `spawn autoscale pause <name>` | Pause reconciliation (keep instances running) |
| `resume` | `spawn autoscale resume <name>` | Resume reconciliation |
| `terminate` | `spawn autoscale terminate <name>` | Terminate group and all instances |
| `set-policy` | `spawn autoscale set-policy <name>` | Set or update queue-depth scaling policy |
| `set-metric-policy` | `spawn autoscale set-metric-policy <name>` | Set metric-based scaling policy |
| `scaling-activity` | `spawn autoscale scaling-activity <name>` | Show recent scaling events |
| `metric-activity` | `spawn autoscale metric-activity <name>` | Show recent metric-based events |
| `add-schedule` | `spawn autoscale add-schedule <name>` | Add a scheduled scaling action |
| `remove-schedule` | `spawn autoscale remove-schedule <name> <schedule-name>` | Remove a scheduled action |
| `list-schedules` | `spawn autoscale list-schedules <name>` | List all scheduled actions |

**Persistent flags** (all subcommands):

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--table` | string | `spawn-autoscale-groups` | DynamoDB table name |
| `--env` | string | `production` | Environment: `production` or `staging` |

**`spawn autoscale launch` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | (required) | Group name |
| `--instance-type` | string | (required) | EC2 instance type |
| `--ami` | string | (required) | AMI ID |
| `--desired-capacity` | int | (required) | Initial desired instance count |
| `--min-capacity` | int | `0` | Minimum instance count |
| `--max-capacity` | int | (desired×2) | Maximum instance count |
| `--spot` | bool | `false` | Use Spot instances |
| `--job-array-id` | string | (auto) | Job array ID |
| `--key-name` | string | | SSH key pair name |
| `--subnet-id` | string | | Subnet ID |
| `--security-groups` | strings | | Security group IDs |
| `--iam-profile` | string | | IAM instance profile |
| `--user-data` | string | | User data script (base64) |
| `--tags` | key=value | | Additional EC2 tags |
| `--scaling-policy` | string | | `queue-depth` or `metric` |
| `--queue-url` | string | | SQS queue URL for queue-depth scaling |
| `--target-messages-per-instance` | int | `10` | Scale up when queue exceeds this per instance |
| `--scale-up-cooldown` | int | `60` | Seconds between scale-up actions |
| `--scale-down-cooldown` | int | `300` | Seconds between scale-down actions |
| `--metric-policy` | string | | Metric name for metric-based scaling |
| `--target-value` | float | | Target metric value |
| `--metric-name` | string | | CloudWatch metric name |
| `--metric-namespace` | string | | CloudWatch metric namespace |
| `--metric-statistic` | string | `Average` | CloudWatch statistic |
| `--metric-period` | int | `300` | Metric evaluation period (seconds) |

**`spawn autoscale update` flags:** `--desired-capacity`, `--min-capacity`, `--max-capacity`

**`spawn autoscale set-policy` flags:** `--scaling-policy`, `--queue-url`, `--queue` (multiple), `--queue-weight`, `--target-messages-per-instance`, `--scale-up-cooldown`, `--scale-down-cooldown`, `--none` (remove policy)

**`spawn autoscale set-metric-policy` flags:** `--metric-policy`, `--target-value`, `--metric-name`, `--metric-namespace`, `--metric-statistic`, `--metric-period`, `--none`

**`spawn autoscale add-schedule` flags:** `--name`, `--schedule` (cron expression), `--desired-capacity`, `--min-capacity`, `--max-capacity`

---

## spawn burst

Launch cloud instances to join a local job array (cloud bursting).

```
spawn burst --job-array-id <id> [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--job-array-id` | string | (required) | Job array ID to join |
| `--job-array-name` | string | | Job array name |
| `--count` | int | `1` | Number of instances to burst |
| `--instance-type` | string | `t3.micro` | Instance type |
| `--spot` | bool | `false` | Use Spot instances |
| `--ami` | string | | AMI ID |
| `--key-name` | string | | SSH key pair name |
| `--subnet-id` | string | | Subnet ID |
| `--security-groups` | strings | | Security group IDs |

---

## spawn slurm

Convert and run Slurm batch scripts on EC2.

```
spawn slurm <subcommand> <script.sbatch>
```

Translates `#SBATCH` directives to spawn flags. Supports `--nodes`, `--ntasks`, `--mem`, `--gres`, `--time`, `--partition`, and others.

| Subcommand | Description |
|------------|-------------|
| `convert <script>` | Convert sbatch script to spawn launch flags |
| `estimate <script>` | Estimate cloud cost for the job |
| `submit <script>` | Convert and launch on EC2 |

**`spawn slurm convert` flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | | Output file for converted parameters |
| `--force-yes` | | bool | `false` | Skip confirmation prompts |

**`spawn slurm submit` flags:** inherits `--spot`

---

## spawn stage

Manage data staging for multi-region sweeps.

```
spawn stage <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `upload` | `spawn stage upload <local-path>` | Upload data to regional S3 buckets |
| `list` | `spawn stage list` | List staged datasets |
| `estimate` | `spawn stage estimate` | Estimate transfer cost savings |
| `delete` | `spawn stage delete <id>` | Delete staged data |

**`spawn stage upload` flags:** `--regions`, `--dest`, `--sweep-id`

---

## spawn cost

View cost breakdown for a parameter sweep.

```
spawn cost breakdown <sweep-id>
```

Shows per-instance resource costs, utilization, and budget status.

---

## spawn dns

Manage DNS records for instances.

```
spawn dns <subcommand>
```

| Subcommand | Description |
|------------|-------------|
| `register` | Register a DNS record for an instance |
| `delete` | Delete a DNS record |
| `list` | List DNS records |

---

## spawn notify

Manage Slack and Teams notification registrations for instances.

```
spawn notify <subcommand>
spawn bot <subcommand>
```

`spawn bot` is an alias for `spawn notify`.

| Subcommand | Description |
|------------|-------------|
| `register` | Register an instance for chat bot control |
| `deregister` | Remove a chat bot registration |
| `enable` | Re-enable bot access for an instance |
| `disable` | Temporarily disable bot access |
| `list` | List bot registrations for a workspace |
| `workspace-add` | Register a Slack/Teams workspace's bot token |
| `workspace-remove` | Remove a workspace registration |
| `workspace-list` | List registered workspaces |

**`spawn notify register` flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--platform` | string | (required) `slack` or `teams` |
| `--user` | string | User email address (resolved to platform user ID) |
| `--user-id` | string | Platform user ID (alternative to `--user`) |
| `--workspace-id` | string | Slack workspace ID or Teams tenant ID |
| `--instance-id` | string | Instance to register |
| `--nickname` | string | Display name for the instance in chat |
| `--connect-code` | string | Self-registration code from `/spore connect` |
| `--role-arn` | string | Cross-account IAM role ARN |

**`spawn notify workspace-add` flags:** `--platform`, `--workspace-id`, `--bot-token`, `--signing-secret`

---

## spawn alerts

Manage alerts for sweeps and schedules.

```
spawn alerts <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `create` | `spawn alerts create <sweep-id>` | Create a new alert |
| `list` | `spawn alerts list` | List all alerts |
| `delete` | `spawn alerts delete <alert-id>` | Delete an alert |

**`spawn alerts create` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--on-complete` | bool | `false` | Alert on completion |
| `--on-failure` | bool | `false` | Alert on failure |
| `--cost-threshold` | float | `0` | Alert when spend exceeds this amount |
| `--long-running` | int | `0` | Alert after this many hours |
| `--instance-failed` | bool | `false` | Alert when an instance fails |
| `--email` | string | | Email destination |
| `--slack` | string | | Slack webhook URL |
| `--sns` | string | | SNS topic ARN |
| `--webhook` | string | | Webhook URL |

---

## spawn validate

Validate compliance and infrastructure configuration.

```
spawn validate [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--nist-800-171` | string | | Validate NIST 800-171 compliance |
| `--nist-800-53` | string | | Validate NIST 800-53 (`low`, `moderate`, `high`) |
| `--infrastructure` | bool | `false` | Validate infrastructure resources |
| `--instance-id` | string | | Specific instance to validate |
| `--region` | string | | AWS region |
| `--output` | string | `text` | Output format: `text`, `json` |

---

## spawn pipeline

Manage multi-stage compute pipelines.

```
spawn pipeline <subcommand>
```

Chains stages together: when one stage completes (via `--on-complete`), the next launches automatically with its results.

See the [pipelines guide](/guides/pipelines) for full workflow documentation.

---

## spawn plugin

Manage plugins for instance customization at launch time.

```
spawn plugin <subcommand>
```

| Subcommand | Description |
|------------|-------------|
| `install` | Install a plugin |
| `list` | List installed plugins |
| `status` | Show plugin status |
| `remove` | Remove a plugin |
| `validate` | Lint one or more `plugin.yaml` files offline (no install): `spawn plugin validate <path>...` |

See the [plugins guide](/guides/plugins) for usage.

---

## spawn instance-config

Read or write runtime configuration on a running instance via SSH.

```
spawn instance-config <instance-id-or-name> <action> [key] [value]
spawn config <instance-id-or-name> <action> [key] [value]
```

`spawn config` is an alias. Actions: `get`, `set`, `list`.

**Examples:**
```sh
spawn config my-job list
spawn config my-job get idle-timeout
spawn config my-job set idle-timeout 2h
```

---

## spawn team

Manage team-shared instance access.

```
spawn team <subcommand>
```

See the [teams guide](/guides/teams-setup) for full workflow documentation.

---

## spawn availability

Display historical launch success/failure statistics for an instance type.

```
spawn availability --instance-type <type> [--regions <regions>]
```

**Flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--instance-type` | string | (required) Instance type to check |
| `--regions` | string | Comma-separated regions |

---

## spawn defaults

Manage launch defaults stored in `~/.spawn/config.yaml`.

```
spawn defaults <subcommand>
```

---

## spawn ami

Manage spawn-managed AMIs.

```
spawn ami <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `list` | `spawn ami list` | List spawn-managed AMIs |
| `create` | `spawn ami create <instance-id-or-name>` | Create an AMI from a running instance |

**`spawn ami list` flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--region` | string | AWS region (default: current region) |
| `--stack` | string | Filter by `spawn:stack` tag |
| `--version` | string | Filter by `spawn:version` tag |
| `--arch` | string | Filter by architecture: `x86_64`, `arm64` |
| `--gpu` | string | Filter by GPU support: `true`, `false` |
| `--deprecated` | bool | Include deprecated AMIs |

**`spawn ami create` flags:**

| Flag | Type | Description |
|------|------|-------------|
| `--name` | string | (required) Name for the AMI |
| `--description` | string | Description |
| `--tag` | strings | Tags in `key=value` format (repeatable) |
| `--reboot` | bool | Reboot instance before creating (default: no-reboot) |
| `--wait` | bool | Wait for AMI to become available |

---

## spawn fsx

Manage spawn-managed FSx Lustre filesystems.

```
spawn fsx <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `list` | `spawn fsx list` | List all spawn-managed FSx filesystems |
| `info` | `spawn fsx info <filesystem-id>` | Show filesystem details and cost estimate |
| `delete` | `spawn fsx delete <filesystem-id>` | Delete a filesystem |

**`spawn fsx delete` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--export-first` | bool | `false` | Export data to S3 before deleting |
| `--yes` | bool | `false` | Skip confirmation prompt |

---

## spawn app

Launch and manage catalog applications via NICE DCV browser streaming.

```
spawn app <subcommand>
```

| Subcommand | Usage | Description |
|------------|-------|-------------|
| `list` | `spawn app list` | List all streamable applications in the catalog |
| `launch` | `spawn app launch <app-name>` | Launch a catalog application in a browser tab |

**`spawn app list`** prints each app's name, description, GPU requirement, recommended instance families, and license type.

**`spawn app launch` flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--name` | string | `<app>-<timestamp>` | Session name |
| `--instance-type` | string | (catalog default) | Override instance type |
| `--region` | string | (AWS config) | AWS region |
| `--spot` | bool | `false` | Use Spot pricing |
| `--ttl` | string | | Hard termination deadline (e.g. `8h`) |
| `--idle-timeout` | string | (catalog default) | Stop when DCV has no clients for this duration |
| `--no-open` | bool | `false` | Write session file but do not open browser automatically |

After launch, `spawn connect <name>` reconnects to an existing session — waking the instance if stopped and opening the browser tab.

---

## spawn version

Print version, build date, and git commit.

```
spawn version
```

---

## spawn completion

Generate shell completion scripts.

```
spawn completion <shell>
```

**Supported shells:** `bash`, `zsh`, `fish`, `powershell`

**Setup examples:**
```sh
# bash
spawn completion bash > /etc/bash_completion.d/spawn

# zsh
spawn completion zsh > "${fpath[1]}/_spawn"

# fish
spawn completion fish | source
```
