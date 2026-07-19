## `truffle list`

List available EC2 instance types, families, or sizes.

Examples:
  truffle list --family
  truffle list --sizes
  truffle list --region us-east-1

```
truffle list [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--family` |  | bool |  | List instance families (e.g., m5, c5, r5) |
| `--region` |  | string | `us-east-1` | Region to query for listing (default: us-east-1) |
| `--sizes` |  | bool |  | List available sizes (e.g., large, xlarge, 2xlarge) |

