## `truffle find`

Find EC2 instance types using natural language, glob patterns, or regex.

Auto-detects query type:
  - Patterns: m7i*, c[6-8]i.large, g5.* → pattern matching
  - Natural language: "graviton 8 cores 32gb" → spec-based search

Understands:
  - CPU vendors: intel, amd, graviton, nvidia
  - Processors: emerald rapids, sapphire rapids, ice lake, genoa, turin, milan
  - GPUs: h200, h100, a100, b200, b300, l40s, l4, a10g, t4, rtx, inferentia, trainium
  - Specs: 8 cores, 8 physical cores, 32gb, 4 gpus
  - Sizes: tiny, small, medium, large, huge
  - Architecture: x86_64, arm64
  - Network: efa, 10gbps, 25gbps, 50gbps, 100gbps, 200gbps, 400gbps
  - Sort hints: cheap/cheapest, fast/fastest, newest/latest

Examples:
  truffle find "m7i*"                         (glob pattern)
  truffle find "c[6-8]i.large"                (regex pattern)
  truffle find graviton                       (vendor search)
  truffle find "turin 32 cores 64gb" --exact  (exact spec match)
  truffle find "8 physical cores 32gb"        (physical core count)
  truffle find "cheap graviton 8 cores"       (sorted by price)
  truffle find nvidia                         (all NVIDIA GPU instances)
  truffle find "h100 efa"                     (GPU + network)

```
truffle find <query> [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--app` |  | string |  | Application name from catalog (e.g. paraview, igv) |
| `--exact` |  | bool |  | Match exact vCPU and memory values instead of minimum |
| `--pick-first` |  | bool |  | Output only the top result's instance type (useful for piping to spawn) |
| `--service` |  | string | `ec2` | Instance namespace to search: ec2 or sagemaker (ml.* types) |
| `--show-query` |  | bool |  | Show parsed query details |
| `--show-quota` |  | bool |  | Show the per-type training-job quota (SageMaker only) |
| `--skip-azs` |  | bool |  | Skip availability zone lookup (faster) |
| `--timeout` |  | duration | `5m0s` | Timeout for AWS API calls |

