---
description: "Truffle finds and compares EC2 instance types."
---

# Truffle <span class="doc-badge beginner">Beginner</span> <span class="doc-badge stable">Stable</span>

**What it is.** Truffle finds and compares EC2 instance types. It's read-only — it
never launches anything.

**When to use it.** Any time *before* a launch, to answer "what should I run and
what will it cost?" — discover a family, filter by exact specs, compare Spot
prices, and confirm your quota so a launch doesn't fail after you've waited.

**First commands:**

```sh
truffle find "amd genoa 64gb"          # discover a family in plain language
truffle search "m8a.*" --min-vcpu 16   # filter by exact specs
truffle spot m8a.4xlarge               # compare Spot prices across regions
truffle quotas --regions us-east-1 --family M   # confirm you can launch it
truffle az p5.48xlarge                 # check per-AZ availability
```

::: tip Which command? find vs search vs spot vs quotas
- **`find`** — you know what you need *in human terms* ("amd genoa 64gb"). Put specs in the query string; filter flags like `--min-vcpu` don't apply here.
- **`search`** — you know the *exact technical filters*: `search "m8a.*" --min-vcpu 16 --min-memory 64`.
- **`spot`** — you've picked a type and want to compare purchase options / regions.
- **`quotas`** — run immediately before launch, so you don't wait for capacity you're not allowed to use.

`find` and `search` are **not** synonyms — see [Common mistakes](#common-mistakes).
:::

## Install

```sh
brew install spore-host/tap/truffle
```

## AWS profile & account

Like the rest of the suite, truffle honors the shared spore.host config: a global `--profile` (and `--account` guard), the `SPORE_PROFILE`/`AWS_PROFILE` env vars, and the `[spore]` table of `~/.config/spore/config.toml`, resolved **flag > env > file > default**. An unset profile uses the ambient AWS credential chain. Region is per-request via `--regions`/`--region`. See [AWS Authentication](/guides/aws-auth) for the full model, and the [command reference](/tools/reference/truffle#global-flags) for every global flag.

## Sub-commands

Truffle has distinct sub-commands for different tasks. They are **not interchangeable** — flags available on one command may not exist on another.

### `truffle find` — natural language search

Discover instance families using plain language. Understands processor names, GPU models, network capabilities, and size descriptions.

```sh
truffle find "epyc genoa"           # AMD EPYC Genoa (4th gen)
truffle find "h100 8gpu efa"        # NVIDIA H100 with EFA networking
truffle find "graviton large"       # ARM64 Graviton, large size class
truffle find "sapphire rapids 32 cores"
truffle find "milan 64gb"
```

Include specs **in the query string** — `truffle find` does not accept `--min-vcpu` or `--min-memory`:
```sh
truffle find "epyc genoa 16 cores"      # ✅ spec in query
truffle find "epyc genoa" --min-vcpu 16 # ❌ --min-vcpu not available on find
```

Flags:
- `--skip-azs` — faster, skip AZ lookup
- `--regions` — limit to specific regions
- `--app <name>` — find instances suitable for a catalog application

### `truffle search` — pattern search with filters

Search by instance type name pattern (wildcards and regex). Supports numeric filters.

```sh
truffle search "m8a.*"                              # all m8a sizes
truffle search "m8a.*" --min-vcpu 16               # ✅ --min-vcpu works here
truffle search "m8a.*" --min-vcpu 16 --min-memory 64
truffle search "c7a.*" --architecture x86_64
truffle search "g5.*" --skip-azs
```

The pattern is anchored — it must match the full instance type name. Wildcards (`*`, `?`) are supported.

Flags: `--min-vcpu`, `--min-memory`, `--architecture`, `--family`, `--show-price`, `--pick-first`, `--skip-azs`

### `truffle spot` — current Spot prices

Get live Spot prices for a specific instance type across regions and AZs.

```sh
truffle spot m8a.4xlarge
truffle spot "m7a.*" --sort-by-price --active-only
truffle spot g5.xlarge --regions us-east-1,us-west-2 --show-savings
```

### `truffle quotas` — service quota check

Check vCPU quotas before launching to avoid capacity errors.

```sh
truffle quotas --regions us-east-1
truffle quotas --family Standard --regions us-east-1   # M, C, R, T instances
truffle quotas --family P --regions us-east-1          # P-family GPU instances
truffle quotas --service sagemaker --family g5         # SageMaker ml.g5.* quotas
truffle quotas --family Standard --request             # generate increase commands
```

**Instance family codes:**

| Code | Instances |
|------|-----------|
| `Standard` | A, C, D, H, I, M, R, T, Z (general purpose) |
| `G` | g4dn, g5, g6 (graphics/GPU) |
| `P` | p3, p4, p5 (GPU training) |
| `Inf` | inf1, inf2 (Inferentia) |
| `Trn` | trn1 (Trainium) |

### `truffle capacity` — capacity reservations you own

Check existing On-Demand Capacity Reservations and Capacity Blocks **already in your account**.

```sh
truffle capacity
truffle capacity --gpu-only
truffle capacity --instance-types p5.48xlarge,p4d.24xlarge
truffle capacity --blocks                              # Capacity Blocks you already own
```

### `truffle capacity-blocks` — discover purchasable Capacity Blocks

Find purchasable [EC2 Capacity Block for ML](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/capacity-blocks.html) **offerings** — "what can I reserve?" (read-only; queries `DescribeCapacityBlockOfferings`). This is distinct from `truffle capacity --blocks`, which lists blocks you *already* own.

```sh
truffle capacity-blocks --instance-type p5.48xlarge --count 1 --duration-hours 24
truffle capacity-blocks --instance-type p5.48xlarge --count 2 --duration-hours 48 \
  --region us-east-1 --start-after 2026-07-01T00:00:00Z
```

Each offering shows its **id** (what `spawn capacity-block purchase` reserves), instance type/count, AZ, start/end, duration, and up-front price. `--instance-type` and `--duration-hours` are required. This is step 1 of the Capacity Block flow — see [Capacity Blocks for ML](#capacity-blocks-for-ml) below.

## Typical workflow: find → search → spot → check quota → launch

```sh
# 1. Discover the instance family
truffle find "epyc genoa"

# 2. Browse sizes within that family (with spec filters)
truffle search "m8a.*" --min-vcpu 16 --min-memory 64

# 3. Check current Spot prices
truffle spot m8a.4xlarge --sort-by-price --active-only

# 4. Verify you have quota (m8a is Standard family)
truffle quotas --family Standard --regions us-east-1

# 5. Launch
spawn launch my-job --instance-type m8a.4xlarge --spot --ttl 4h
```

## Piping to spawn

Use `--pick-first` to get a single instance type name for piping:

```sh
spawn launch my-job \
  --instance-type $(truffle search "m8a.*" --min-vcpu 16 --pick-first) \
  --spot --ttl 4h
```

## Capacity Blocks for ML

A Capacity Block reserves scarce GPU capacity (e.g. p5.48xlarge) for a future window. The flow spans all three tools — truffle discovers, spawn buys, lagotto launches:

```sh
# 1. truffle — find a purchasable offering (read-only)
truffle capacity-blocks --instance-type p5.48xlarge --count 1 --duration-hours 24

# 2. spawn — purchase it (billed up front, NON-REFUNDABLE; three typed
#    confirmations, interactive only; --dry-run to preview)
spawn capacity-block purchase <offering-id> --instance-type p5.48xlarge \
  --count 1 --duration-hours 24 --region us-east-1

# 3. lagotto — launch into it at the reserved start time
lagotto launch --at <block-start> --az <block-az> --spawn-config block.yaml
```

Truffle stays read-only throughout — the purchase (a real-money, non-refundable write) lives in spawn behind its confirmation gates.

## Common mistakes

- **Treating `find` and `search` as synonyms.** `find` is natural-language (specs go *in the query*); `search` is pattern + flags (`--min-vcpu`). Filter flags don't exist on `find`.
- **Confusing region and AZ.** A type available in a region generally can still be unplaceable in a specific AZ — use `truffle az` and pin with `--az` when it matters.
- **Assuming quota means capacity.** `truffle quotas` shows *permission* to launch, not *availability right now*. You can be within quota and still get `InsufficientInstanceCapacity` — that's what [Lagotto](/tools/lagotto) is for.

See [Troubleshooting & common mistakes](/reference/troubleshooting) for the full list.

## How it connects

Truffle is the read-only front of the workflow: it tells you *what to run*.
[Spawn](/tools/spawn) takes that and launches it (pipe with `--pick-first`, above).
When the type you want has no capacity right now, [Lagotto](/tools/lagotto) waits
for it. Truffle never launches or spends money.

## Full command reference

→ [truffle command reference](/tools/reference/truffle)
