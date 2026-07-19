## `truffle search`

> **Deprecated:** use 'find' instead — it auto-detects patterns vs natural language queries

Search for instance types across AWS regions.

Supports glob patterns (m7i*, c7?) and regex (c[6-8]i\.large, (p4d|p5)\..*) for flexible matching.

```
truffle search [instance-type-pattern] [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--architecture` |  | string |  | Filter by architecture (x86_64, arm64, i386) |
| `--family` |  | string |  | Filter by instance family (e.g., m5, c5) |
| `--min-memory` |  | float64 |  | Minimum memory in GiB |
| `--min-vcpu` |  | int |  | Minimum number of vCPUs |
| `--nested-virtualization` |  | bool |  | Only types supporting nested virtualization (KVM/Hyper-V in-instance) |
| `--pick-first` |  | bool |  | Output only the top result's instance type (useful for piping to spawn) |
| `--service` |  | string | `ec2` | Instance namespace to search: ec2 or sagemaker (ml.* types) |
| `--show-price` |  | bool |  | Show on-demand pricing (uses static pricing data) |
| `--show-quota` |  | bool |  | Show the per-type training-job quota (SageMaker only) |
| `--skip-azs` |  | bool |  | Skip availability zone lookup (faster but less detailed) |
| `--timeout` |  | duration | `5m0s` | Timeout for AWS API calls |

