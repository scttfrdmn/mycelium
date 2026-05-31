# Spawn

Spawn launches EC2 instances and manages their full lifecycle. It provisions the spored daemon on each instance, which handles auto-termination, idle detection, DNS, and lifecycle notifications independently of your laptop.

## Install

```sh
brew install spore-host/tap/spawn
```

## Core commands

### `spawn` / `spawn launch`

Launch an instance. With no arguments, the interactive wizard runs:

```sh
spawn
```

With flags:

```sh
spawn launch \
  --name my-instance \
  --instance-type g5.xlarge \
  --region us-east-1 \
  --ttl 8h
```

### `spawn list`

List your running (or all) instances:

```sh
spawn list
spawn list --state all
spawn list --region us-east-1
```

### `spawn status`

Detailed status for one instance:

```sh
spawn status my-instance
spawn status i-0a1b2c3d4e5f
spawn status my-instance -o json       # machine-readable
spawn status my-instance --check-complete   # exit 0=complete 1=failed 2=running 3=error
```

**`--check-complete`** polls the instance's completion file and exits with a
standardized code, so scripts can wait for a workload to finish (v0.36.6+):

| Exit | Meaning |
|------|---------|
| 0 | Complete — completion file present |
| 1 | Failed — completion file reports a failure status |
| 2 | Running — completion file not yet present |
| 3 | Error — instance unreachable or status undeterminable |

```sh
# Wait for a workload to finish
while spawn status my-job --check-complete; [ $? -eq 2 ]; do sleep 30; done
```

**JSON schema** (`-o json`) — key fields returned:

| Field | Type | Description |
|-------|------|-------------|
| `instance_id` | string | EC2 instance ID |
| `name` | string | Instance name tag |
| `state` | string | `running`, `stopped`, `terminated`, … |
| `public_ip` | string | Public IPv4 address |
| `instance_type` | string | EC2 instance type |
| `region` | string | AWS region |
| `ttl` | string | Remaining TTL (e.g. `3h25m`) |
| `on_complete` | string | Action on completion |
| `tags` | object | All `spawn:*` tags as key-value map |

### `spawn stop` / `spawn hibernate` / `spawn start`

```sh
spawn stop my-instance              # stop (billing pauses, data preserved)
spawn hibernate my-instance         # hibernate to disk (saves RAM state)
spawn start my-instance             # start stopped or hibernated instance
```

`stop` and `hibernate` preserve EBS volumes. To permanently destroy an instance, use `terminate`.

### `spawn terminate`

Permanently terminate an instance — destroys the instance and its EBS volumes
(unlike `stop`/`hibernate`). Irreversible, so it confirms by default:

```sh
spawn terminate my-instance              # prompts, then terminates
spawn terminate my-instance -y           # skip confirmation
spawn terminate --job-array-name workers # terminate a whole job array
```

### `spawn extend`

Update the TTL on a running instance:

```sh
spawn extend my-instance 4h     # extend by 4 hours from now
```

### `spawn connect`

Open an interactive SSH session, or run a command and return. The command is wrapped in `bash -c` on the remote side, so compound operators and background jobs (`&`) work correctly:

```sh
spawn connect my-instance                                                        # interactive
spawn connect my-instance -- 'tail -20 /tmp/run.log'                            # one-shot
spawn connect my-instance -- 'cmd1 && cmd2'                                     # compound
spawn connect my-instance -- 'nohup bash /tmp/run.sh > /tmp/run.log 2>&1 &'    # background
spawn connect my-instance -- 'aws s3 cp s3://bucket/run.sh /tmp/ && bash /tmp/run.sh &'
```

When multiple instances share a name, `spawn connect` prefers the running one. Stopped or hibernated instances are automatically started before connecting — use `--no-start` to prevent this.

### `spawn defaults`

Manage default launch settings:

```sh
spawn defaults set slack-workspace T03NE3GTY
spawn defaults set idle-timeout 1h
spawn defaults set active-processes rsession
spawn defaults list
spawn defaults unset active-processes
```

Defaults are stored in `~/.spawn/config.yaml` and apply to every launch unless overridden. See [Configuration](/reference/configuration).

### `spawn notify`

Register instances and users for Slack/Teams control:

```sh
spawn notify workspace-add ...
spawn notify register ...
spawn notify enable ...
spawn notify list ...
```

See [Slack Setup](/guides/slack-setup) or [Teams Setup](/guides/teams-setup) for the full walkthrough.

## Key concepts

**TTL** — every instance has an absolute termination deadline: `launch_time + TTL`. When it fires, the instance terminates. The deadline is stored in a tag at launch and is **never reset** by stop/wake cycles — it keeps counting even while the instance is stopped. `spawn extend` pushes the deadline forward, not from now.

**Idle timeout** — spored monitors CPU, network, disk, GPU, sessions, and configured process names. When all signals indicate inactivity for the configured duration, the instance **stops** (or hibernates with `--hibernate-on-idle`). The idle timer **resets** every time the instance wakes. Idle timeout never terminates — only TTL does that.

**Spored** — a small daemon that runs on the instance, enforces the TTL deadline, detects idleness, registers DNS, and sends lifecycle notifications. Installed automatically at launch.

**Pre-stop hooks** — a shell command that runs before any lifecycle-triggered stop or termination. Use it to save checkpoints, sync output to S3, or notify downstream systems.

**Job arrays** — `spawn launch --count N` launches N identical instances. Each instance gets a set of environment variables so it knows its role:

| Variable | Description |
|----------|-------------|
| `JOB_ARRAY_ID` | Unique array ID (UUID) |
| `JOB_ARRAY_NAME` | Array name (from `--job-array-name`) |
| `JOB_ARRAY_SIZE` | Total instances in the array |
| `JOB_ARRAY_INDEX` | Zero-based index of this instance (0 … N-1) |

Example — shard a dataset across 8 instances:
```bash
spawn launch data-proc --count 8 --instance-type c6a.xlarge --ttl 2h
# On each instance:
# CHUNK=$((total_chunks / JOB_ARRAY_SIZE))
# START=$((JOB_ARRAY_INDEX * CHUNK))
# process_data --start $START --count $CHUNK
```

**`--region` vs `--regions`** — `spawn` uses `--region` (singular, one value) since a launch targets a single region. `truffle` uses `--regions` / `-r` (plural, comma-separated) since it searches across multiple regions at once. When piping `truffle` output to `spawn`, use the single region from `truffle`'s result:
```bash
region=$(truffle spot c6a.xlarge --sort-by-price --pick-first | jq -r .region)
spawn launch my-job --instance-type c6a.xlarge --region "$region"
```

::: tip
See [TTL vs idle timeout](/reference/configuration#ttl-vs-idle-timeout-how-they-interact) for a complete explanation with a worked timeline.
:::

## Programmatic access

Use spawn from Python scripts, notebooks, or FastAPI backends via the [Python SDK](/guides/python-sdk):

```python
import spore

# List running instances
instances = spore.spawn.list()
for inst in instances:
    print(inst.name, inst.state, inst.ttl)

# Poll status until complete
inst = spore.spawn.status("my-job")
inst.wait("terminated")
```

Or poll `spawn status --check-complete` and branch on its exit code (works for a
single instance and, via `spawn sweep status --check-complete`, for sweeps):
```bash
# Exit 0=complete, 1=failed, 2=running, 3=error
while spawn status my-job --check-complete; [ $? -eq 2 ]; do sleep 30; done
```

## Full command reference

→ [spawn command reference](/tools/reference/spawn)
