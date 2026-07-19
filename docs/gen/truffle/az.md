## `truffle az`

Search for instance types with an availability zone-first perspective.
This command prioritizes showing which specific AZs support each instance type,
making it ideal for multi-AZ deployments and capacity planning.

Examples:
  # Find which AZs have m7i.large
  truffle az m7i.large

  # Search in specific AZs only
  truffle az m7i.large --az us-east-1a,us-east-1b

  # Find instances available in at least 3 AZs per region
  truffle az "m8g.*" --min-az-count 3

  # Show AZ availability summary
  truffle az "c7i.xlarge" --output json

```
truffle az [instance-type-pattern] [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--az` |  | stringSlice |  | Filter by specific availability zones (e.g., us-east-1a,us-west-2b) |
| `--min-az-count` |  | int |  | Minimum number of AZs required per region |
| `--regions-only` |  | bool |  | Show only regions that meet AZ count requirement |
| `--timeout` |  | duration | `5m0s` | Timeout for AWS API calls |

