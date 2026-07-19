## `lagotto deploy`

Stand up lagotto's hosted capacity poller (DynamoDB, SNS, Lambda, EventBridge
Scheduler) in your OWN AWS account, so watches are serviced server-side — armed
once, then hands-off — instead of depending on a foreground 'poll --daemon' that
dies when your laptop sleeps.

It downloads the published capacity-poller Lambda artifact for the given
--version, uploads it to a bucket in your account, and deploys the embedded
CloudFormation stack. The poller schedule deploys DISABLED; the first 'lagotto
watch' enables it (and the poller self-disables when no active watches remain).

Use --teardown to delete the stack.

```
lagotto deploy [flags]
```

**Flags:**

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--environment` |  | string | `production` | Environment tag (production, staging, development) |
| `--lambda-bucket` |  | string |  | S3 bucket for the Lambda artifact (default: lagotto-lambda-&lt;account&gt;-&lt;region&gt;, created if absent) |
| `--region` |  | string |  | AWS region (default: from your AWS config) |
| `--stack-name` |  | string | `lagotto` | CloudFormation stack name |
| `--teardown` |  | bool |  | Delete the stack instead of deploying it |
| `--version` |  | string | `dev` | lagotto release version to pull the poller Lambda from |
| `--yes` | `-y` | bool |  | Skip the confirmation prompt |

