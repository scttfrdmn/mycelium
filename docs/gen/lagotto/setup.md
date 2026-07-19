## `lagotto setup`

Create the DynamoDB tables lagotto uses to store watches and match history
(lagotto-watches and lagotto-match-history by default; override with
--watches-table / --history-table).

This is idempotent — existing tables are left untouched. You don't normally need
to run it: 'lagotto watch' creates the tables automatically on first use. Run it
explicitly when you want to provision the backend ahead of time or confirm what
will be created.

```
lagotto setup
```

