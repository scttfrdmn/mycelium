# spore-bot — OpenTofu module

The **first IaC in the umbrella**. It reconciles the (previously hand-deployed)
`spore-bot` Lambda and its dedicated execution role under OpenTofu, as the
reference pattern other hosted components follow. Everything else in the umbrella
is still imperative `setup-*.sh`; this is the start of migrating off that.

## What it manages

- `aws_iam_role.spore_bot` — **`spore-bot-role`**, least-privilege, dedicated.
  (Replaces the prior wrong setup where spore-bot ran under
  `prism-bot-PrismBotFunctionRole`.) DynamoDB RW on the three `spore-bot-*`
  tables, Lambda self-invoke, `SpawnBotCrossAccount` assume, and scoped logs.
- `aws_lambda_function.spore_bot` — the function's **shape** (role, runtime,
  arm64, memory, timeout).
- `aws_lambda_function_url.spore_bot` + public invoke permission — the Function
  URL that Discord's interactions endpoint and instances' `spawn:notify-url`
  depend on. It is deterministic from function name + account + region, so it is
  preserved as long as the function keeps its name.

## What it deliberately does NOT manage

- **Function code** — deployed out-of-band via `lambda/spore-bot` →
  `make deploy` / `update-function-code`. Tofu `ignore_changes` covers all code
  attributes so a deploy is never reverted.
- **Environment variables** — they hold secrets (OAuth, Teams, Discord public
  key). Tofu ignores `environment`, so secrets stay out of source and aren't
  clobbered. Set them with `aws lambda update-function-configuration` (or a
  secrets store later).

## State

Local state, gitignored (`*.tfstate`). A remote backend (S3 + DynamoDB lock) is
a follow-up once more components are migrated. Never commit state or
`placeholder.zip`.

## How it was imported (runbook)

The live resources were created imperatively, then imported so Tofu adopts them
with **zero functional diff** (only additive `managedby=opentofu` tags):

```sh
export AWS_PROFILE=spore-host-infra
tofu init
tofu import aws_iam_role.spore_bot spore-bot-role
tofu import 'aws_iam_role_policy_attachment.basic_execution' 'spore-bot-role/arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole'
tofu import aws_iam_role_policy.spore_bot 'spore-bot-role:spore-bot-permissions'
tofu import aws_lambda_function.spore_bot spore-bot
tofu import aws_lambda_function_url.spore_bot spore-bot
tofu import aws_lambda_permission.url_public 'spore-bot/FunctionURLAllowPublicAccess'
tofu plan      # must be 0 to add / 0 to destroy (only tag changes)
tofu apply
```

## Day-to-day

- Change the role/permissions/function shape here → `tofu plan` → `tofu apply`.
- Ship new code: `cd ../../../lambda/spore-bot && make deploy` (unchanged).
- `tofu plan` should stay clean; a non-tag diff means something drifted
  out-of-band and is worth investigating.

## Cleanup (done 2026-06-14)

- ✅ Deleted the empty stuck `spore-bot` CloudFormation stack (was
  `REVIEW_IN_PROGRESS`, owned nothing).
- ✅ Removed the orphaned `SporeBotSelfInvoke` inline policy from the prism-bot
  role — spore-bot now has its own `spore-bot-role`, so the prism↔spore coupling
  is fully severed.
