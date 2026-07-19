## `lagotto setup`

Provision lagotto's backend: the DynamoDB tables it uses to store watches and
match history (lagotto-watches and lagotto-match-history by default; override with
--watches-table / --history-table), and — if the hosted poller has been deployed
('lagotto deploy') — the runtime IAM policy that lets the poller spawn/hold/submit.

The table creation is idempotent (existing tables are left untouched) and normally
automatic: 'lagotto watch' creates the tables on first use. Run 'setup' explicitly
to provision the backend ahead of time, or — importantly — after 'lagotto deploy'
to grant the poller its permissions. 'deploy' creates only a minimal execution
role (so the runtime Lambda can never self-escalate); 'setup', run by you, attaches
the spawn/hold/SageMaker/scheduler policy. Until then the poller can only notify.
If the poller role doesn't exist yet, setup creates the tables and prints a
next-step note instead of failing.

```
lagotto setup
```

