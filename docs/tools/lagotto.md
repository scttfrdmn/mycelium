# Lagotto

Lagotto watches for EC2 instance capacity and acts when it appears. It runs as a serverless Lambda function — no always-on server required. Configure a watch, deploy, and Lagotto polls on a schedule until it can launch what you asked for.

There is no AWS API that reports "capacity is available right now" — the only true test is an actual launch. So for `spawn` watches **the launch attempt is the capacity test**: Lagotto tries to launch, and if AWS returns `InsufficientInstanceCapacity` it simply keeps the watch active and tries again on the next poll. It does the tedious retrying for you instead of you sitting there re-running a launch by hand, until it succeeds or the watch's TTL expires.

## Install

```sh
brew install spore-host/tap/lagotto
```

## The problem it solves

Some instance types — particularly high-demand GPU families like p5.48xlarge or trn1.32xlarge — have intermittent availability. You want the instance, but there's none available right now. Without Lagotto, your options are: keep trying manually, or write your own polling loop.

Lagotto automates the waiting.

## Core commands

### `lagotto watch`

Create a watch for an instance type pattern in one or more regions:

```sh
# Watch for any p5 instance and notify when available
lagotto watch "p5.*" --ttl 7d

# Limit to specific regions
lagotto watch "p5.48xlarge" --regions us-east-1,us-west-2

# Watch and notify via email or webhook
lagotto watch "p5.48xlarge" --action notify \
  --notify email:you@example.com

# Watch and auto-launch when capacity appears
lagotto watch "g5.xlarge" --action spawn \
  --spawn-config my-job.yaml

# Watch for Spot capacity under a price ceiling
lagotto watch "p4d.24xlarge" --spot --max-price 10.00

# Watch for SageMaker ml.* capacity (EC2-family proxy — see below)
lagotto watch "ml.g5.2xlarge" --service sagemaker \
  --notify email:you@example.com
```

#### Watching SageMaker capacity (`--service sagemaker`)

SageMaker Training/Processing jobs can fail with `CapacityError` even when your
quota is sufficient. AWS exposes **no** read-only SageMaker capacity API, so
`--service sagemaker` watches the **correlated EC2 family as a proxy**
(`ml.g5.2xlarge` → `g5.2xlarge`). EC2 `g5` availability is a *hint* that
SageMaker `ml.g5` capacity is likely available — but SageMaker is a separate
managed pool, so it is not a guarantee.

Because Lagotto cannot submit your SageMaker job for you, SageMaker watches are
**notify-only** (`--action spawn`/`hold` are rejected). When the proxy fires, the
notification tells you it's worth **retrying your SageMaker job** — and to leave
the watch running and retry again on the next notification if the job still hits
`CapacityError`.

### `lagotto list`

```sh
lagotto list              # active watches only
lagotto list --all        # include matched, failed, expired, and cancelled
lagotto list --output json
```

### `lagotto status`

```sh
lagotto status <watch-id>
```

### `lagotto cancel`

```sh
lagotto cancel <watch-id>
```

### `lagotto extend`

```sh
lagotto extend <watch-id> --ttl 48h
```

### `lagotto history`

```sh
lagotto history                        # all your matches
lagotto history --watch-id <watch-id>  # one watch
```

### `lagotto poll`

Manually trigger one polling cycle (useful for testing):

```sh
lagotto poll
```

## Actions

When a watch matches, Lagotto can:

| Action | What happens |
|--------|-------------|
| `notify` | Sends a notification via `--notify` channels (email, webhook, SNS) |
| `spawn` | Launches an instance using the config file given in `--spawn-config` |
| `hold` | Creates a short-lived On-Demand Capacity Reservation to hold the capacity |

SageMaker watches (`--service sagemaker`) support `notify` only.

## Watch lifecycle

A watch is `active` while it's being polled, and ends in one of:

| Status | Meaning |
|--------|---------|
| `matched` | The action succeeded (notified, launched, or held) |
| `failed` | A launch hit a **terminal** error that retrying can't fix (bad AMI/IAM, exhausted quota) |
| `expired` | The watch TTL elapsed before it could act |
| `cancelled` | You cancelled it with `lagotto cancel` |

The watch **TTL is the only time limit** — there is no max-retry count. A
capacity failure (`InsufficientInstanceCapacity`) is *not* terminal: the watch
stays `active` and retries on the next poll until it launches or the TTL runs out.

## How it works

Lagotto deploys as an AWS Lambda function with an EventBridge schedule trigger.
Each tick (default 5 minutes) it pre-filters with `DescribeInstanceTypeOfferings`
and spot pricing to decide which watches are worth acting on — but those are only
hints, not capacity guarantees. For a `spawn` watch the actual launch is the real
test: a capacity failure keeps the watch active to retry, while a terminal failure
marks it `failed`.

The poller is a **self-terminating, per-account singleton**: there is one Lambda
+ schedule per account, every invocation sweeps all active watches, and watches
drop out of the active set as they launch, fail, or expire. When **zero** active
watches remain, the Lambda disables its own schedule — no watches, no Lambda.
Creating a new watch re-arms it.

## Deploy

Lagotto is deployed via CloudFormation — not through the CLI. See the [deployment guide](https://github.com/spore-host/spore-host/blob/main/lagotto/DEPLOYMENT.md) for the full setup: Lambda, EventBridge schedule, DynamoDB tables, and IAM role. Once deployed, the `lagotto` CLI manages watches in that infrastructure.

## Full command reference

→ [lagotto command reference](/tools/reference/lagotto)
