# truffle command reference

## Global flags

All truffle commands inherit these persistent flags.

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--output` | `-o` | string | `table` | Output format: `table`, `json`, `yaml`, `csv` |
| `--regions` | `-r` | strings | (all enabled) | Filter by regions (comma-separated) |
| `--no-color` | | bool | `false` | Disable colorized output |
| `--verbose` | `-v` | bool | `false` | Enable verbose output |
| `--lang` | | string | (system) | Language for output: `en`, `es`, `fr`, `de`, `ja`, `pt` |
| `--no-emoji` | | bool | `false` | Disable emoji in output |
| `--accessibility` | | bool | `false` | Enable accessibility mode (implies `--no-emoji`) |

---

## truffle search

Search for instance types by pattern (wildcard or regex).

```
truffle search <pattern>
```

**Pattern syntax:** Wildcards (`*`, `?`) are supported and automatically converted to regex. The pattern is anchored — it must match the full instance type name.

**Examples:**
```sh
truffle search m7i.xlarge
truffle search "p4d.*"
truffle search "c[6-8]*.large"
truffle search "r7*" --regions us-east-1,us-west-2
truffle search "m7i.*" --min-vcpu 16 --min-memory 64
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--architecture` | string | (any) | Filter by architecture: `x86_64`, `arm64`, `i386` |
| `--family` | string | (any) | Filter by instance family (e.g., `m5`, `c6i`) |
| `--min-vcpu` | int | `0` | Minimum vCPU count |
| `--min-memory` | float | `0` | Minimum memory in GiB |
| `--skip-azs` | bool | `false` | Skip AZ lookup (faster, but hides AZ column) |
| `--show-price` | bool | `false` | Show on-demand pricing |
| `--pick-first` | bool | `false` | Print only the top result's instance type (for piping to spawn) |
| `--timeout` | duration | `5m` | Timeout for AWS API calls |

---

## truffle find

Find instances using natural language queries.

```
truffle find [query...]
```

Understands CPU vendors, processor code names, GPU models, sizes, specs, network requirements, and application names. Multiple words are joined as a single query.

**Examples:**
```sh
truffle find graviton
truffle find "amd epyc genoa"
truffle find "h100 8gpu efa"
truffle find "sapphire rapids 32 cores"
truffle find inferentia
truffle find "100gbps intel"
truffle find --app paraview
```

**Supported keywords:**

| Category | Keywords |
|----------|----------|
| CPU vendor | `intel`, `amd`, `graviton` |
| Processor | `ice lake`, `milan`, `sapphire rapids`, `genoa` |
| GPU | `a100`, `v100`, `h100`, `t4`, `l4`, `inferentia`, `trainium` |
| Size | `tiny`, `small`, `medium`, `large`, `huge` |
| Specs | `8 cores`, `32gb`, `4 gpus` |
| Network | `efa`, `10gbps`, `25gbps`, `50gbps`, `100gbps` |
| Architecture | `x86_64`, `arm64` |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--app` | string | | Application name from the spore.host catalog (e.g., `paraview`, `igv`) |
| `--show-query` | bool | `false` | Print parsed query interpretation before results |
| `--skip-azs` | bool | `false` | Skip AZ lookup (faster) |
| `--timeout` | duration | `5m` | Timeout for AWS API calls |

---

## truffle spot

Find Spot instance availability and current pricing.

```
truffle spot <pattern>
```

Queries AWS Spot price history for all instance types matching the pattern across all enabled regions (or those specified with `--regions`). Prints a price-range summary plus a per-AZ table.

With `--show-savings`, the table adds On-Demand and Savings columns. On-demand rates are fetched live from the AWS Price List API (cached for a day) and fall back to a built-in estimate when the API is unavailable.

The summary block is printed only for the default `table` output; `--output json`, `csv`, and `yaml` emit just the structured data, so they pipe cleanly into `jq` and other tools.

**Examples:**
```sh
truffle spot c6a.xlarge
truffle spot "c7*" --sort-by-price --active-only
truffle spot "g4dn.*" --max-price 1.50 --show-savings
truffle spot p4d.24xlarge --regions us-east-1,us-west-2
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--max-price` | float | `0` | Maximum Spot price per hour (USD); `0` = no limit |
| `--show-savings` | bool | `false` | Show savings percentage vs on-demand price |
| `--sort-by-price` | bool | `false` | Sort results cheapest first |
| `--active-only` | bool | `false` | Only show AZs with active Spot capacity in the lookback window |
| `--lookback-hours` | int | `1` | Hours of price history to query (1–720) |
| `--local-zones` | bool | `false` | Include local zones in results |
| `--pick-first` | bool | `false` | Print only the top result's instance type |
| `--timeout` | duration | `5m` | Timeout for AWS API calls |

---

## truffle az

Search with an availability zone-first perspective.

```
truffle az <pattern>
```

Like `truffle search`, but sorts results by AZ count (most AZs first) and lets you filter by specific AZs or require a minimum AZ count. Useful for multi-AZ deployment planning.

**Examples:**
```sh
truffle az m7i.large
truffle az m7i.large --az us-east-1a,us-east-1b
truffle az "m8g.*" --min-az-count 3
truffle az c7i.xlarge --output json
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--az` | strings | (all) | Filter by specific AZs (comma-separated, e.g., `us-east-1a,us-east-1b`) |
| `--min-az-count` | int | `0` | Minimum AZs required per region |
| `--regions-only` | bool | `false` | Show only regions that meet the `--min-az-count` requirement |
| `--timeout` | duration | `5m` | Timeout for AWS API calls |

**Inherited filters:** `--architecture`, `--family`, `--min-vcpu`, `--min-memory`

---

## truffle capacity

Show On-Demand Capacity Reservations and Capacity Blocks.

```
truffle capacity
```

Queries your AWS account for existing capacity reservations and capacity blocks. Requires AWS credentials.

**Examples:**
```sh
truffle capacity
truffle capacity --available-only
truffle capacity --gpu-only
truffle capacity --instance-types p5.48xlarge,p4d.24xlarge
truffle capacity --blocks
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--instance-types` | strings | (all) | Filter by specific instance types (comma-separated) |
| `--available-only` | bool | `false` | Only show reservations with available capacity |
| `--active-only` | bool | `true` | Only show active reservations |
| `--min-capacity` | int | `0` | Minimum available capacity |
| `--gpu-only` | bool | `false` | Only show GPU/ML instance reservations (p, g, inf, trn families) |
| `--blocks` | bool | `false` | Show Capacity Blocks for ML (training workloads) |
| `--odcr` | bool | `true` | Show On-Demand Capacity Reservations |
| `--timeout` | duration | `5m` | Timeout for AWS API calls |

---

## truffle list

List instance types, families, or sizes from a single region.

```
truffle list [--family | --sizes] [--region <region>]
```

Useful for browsing what's available or building scripts that enumerate instance types.

**Examples:**
```sh
truffle list --family
truffle list --sizes
truffle list --region eu-west-1
```

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--family` | bool | `false` | List instance families (e.g., `m5`, `c6i`, `r7g`) |
| `--sizes` | bool | `false` | List available sizes (e.g., `large`, `xlarge`, `2xlarge`) |
| `--region` | string | `us-east-1` | Region to query |

---

## truffle quotas

Show AWS Service Quotas for EC2 and SageMaker instances.

```
truffle quotas [--regions <regions>] [--family <family>] [--request]
```

Displays current quotas, running usage, and available headroom. Use `--service sagemaker` to check SageMaker `ml.*` instance quotas (processing, training, endpoint, transform). Requires AWS credentials.

**Examples:**
```sh
# EC2 quotas (default)
truffle quotas
truffle quotas --regions us-east-1,us-west-2
truffle quotas --family P --request

# SageMaker ml.* instance quotas
truffle quotas --service sagemaker --regions us-west-2
truffle quotas --service sagemaker --family g5 --regions us-west-2
truffle quotas --service sagemaker --family g5 --request --regions us-west-2
```

**EC2 instance family codes:**

| Code | Instances |
|------|-----------|
| `Standard` | A, C, D, H, I, M, R, T, Z (general purpose) |
| `G` | g4dn, g5, g6 (graphics/GPU) |
| `P` | p3, p4, p5 (GPU training) |
| `Inf` | inf1, inf2 (Inferentia) |
| `Trn` | trn1 (Trainium) |
| `F` | f1 (FPGA) |
| `X` | x1, x2 (memory-optimized) |

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | `ec2` | Service to query: `ec2` or `sagemaker` |
| `--regions` | strings | `us-east-1` | Regions to check (comma-separated) |
| `--family` | string | (all) | EC2: family code (Standard/G/P/Inf/Trn); SageMaker: instance family prefix (e.g. `g5`, `p4d`) |
| `--request` | bool | `false` | Generate `aws service-quotas request-service-quota-increase` commands |

---

## truffle app

Browse and search the spore.host application catalog.

```
truffle app list
```

Lists streamable research applications registered in the spore.host catalog. Use an app name with `truffle find --app <name>` to find suitable instance types.

**Subcommands:**

| Subcommand | Description |
|------------|-------------|
| `list` | List all catalog entries with descriptions |

---

## truffle version

Print version, build date, and git commit.

```
truffle version
```

---

## truffle completion

Generate shell completion scripts.

```
truffle completion <shell>
```

**Supported shells:** `bash`, `zsh`, `fish`, `powershell`

**Setup examples:**
```sh
# bash
truffle completion bash > /etc/bash_completion.d/truffle

# zsh
truffle completion zsh > "${fpath[1]}/_truffle"

# fish
truffle completion fish | source
```
