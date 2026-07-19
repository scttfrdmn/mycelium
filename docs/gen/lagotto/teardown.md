## `lagotto teardown`

Delete the DynamoDB tables lagotto uses (lagotto-watches and
lagotto-match-history by default).

lagotto already tears these down automatically once there are no active watches
and the tables have drained (watches and match history age out via DynamoDB TTL).
Use this to remove them explicitly.

By default it refuses to delete tables that still hold records, so you don't lose
match history; pass --force to delete regardless.

```
lagotto teardown [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--force` |  | bool |  | Delete even if the tables still contain records |

