---
description: "spore.host tools (spawn, truffle, lagotto) act on AWS with your own AWS credentials."
---

# AWS Authentication

spore.host tools (`spawn`, `truffle`, `lagotto`) act on AWS with **your own AWS credentials**. There is no separate spore.host login — if the AWS CLI can reach your account, so can spore.host. This page is the one place that explains how to authenticate, how profiles work, and how your authentication relates to permissions and to what spore.host does on your behalf.

## Sign in with `aws login`

The recommended way to authenticate is **`aws login`** (AWS CLI **v2.34+**), which signs you in and manages **short-lived, auto-refreshing** credentials:

```sh
aws login                     # opens your browser to sign in
aws sts get-caller-identity   # confirm: prints your account + identity
```

That's it — `spawn`, `truffle`, and `lagotto` pick these credentials up automatically through the standard AWS credential chain.

::: tip Why `aws login` over static keys
`aws login` credentials expire and refresh, so there's no long-lived secret to leak. Prefer it. Static access keys (`aws configure`) still work as a fallback — see below — but treat them as legacy/CI-only.
:::

### Static keys (fallback)

If your organization issues static keys instead of federated login:

```sh
aws configure   # Access Key ID, Secret Access Key, region
```

Or export them (e.g. in CI): `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`.

## Profiles and per-tool config

Select a profile per-command or for the session with the standard AWS variable:

```sh
AWS_PROFILE=research spawn launch experiment --instance-type g5.xlarge --ttl 8h
export AWS_PROFILE=research
```

spore.host also has a **suite-wide config layer** (`sporeconfig`) so you can pin a profile/region/account for the spore.host tools specifically, resolved **flag > env > file > default**:

| Layer | Example |
|-------|---------|
| Flag  | `spawn --profile research --region us-west-2 launch …` |
| Env   | `export SPORE_PROFILE=research SPORE_REGION=us-west-2` |
| File  | `~/.config/spore/config.toml` → `[spore]\nprofile = "research"` |

`SPORE_PROFILE`/`SPORE_REGION` override `AWS_PROFILE`/`AWS_REGION` for spore.host tools only, leaving the plain AWS CLI unaffected. `SPORE_ACCOUNT` (or `--account`) records the AWS account you expect, so a tool refuses to act if your credentials resolve to a different account — a guard against launching in the wrong place.

## Authentication vs permissions vs "what runs where"

Three distinct things, easy to conflate:

1. **Authentication** — *who you are.* `aws login` (or a profile/keys) proves your identity to AWS.
2. **Permissions** — *what you're allowed to do.* Your IAM role/user must allow the EC2/IAM actions spawn makes. That's the [minimal IAM policy](/reference/iam-permissions) — attach it to the identity you authenticate as. If a launch fails with `AccessDenied` / `UnauthorizedOperation`, this is what to check.
3. **What acts on your behalf** — spore.host's hosted services (DNS subdomains, Slack/Teams notifications) run in spore.host's own AWS account and are reached over an HTTPS API; the **launched EC2 instances run in *your* account** under *your* credentials. So: you authenticate as you → your IAM policy grants the launch → spawn runs `RunInstances` in your account → the in-instance `spored` daemon manages lifecycle → optional hosted features (DNS/notify) are called out to spore.host's API. (Running everything in your *own* account instead — including the hosted pieces — is [self-hosting](/guides/self-hosting).)

## SSH keys and the instance login user

When spawn launches an instance it sets you up to **log in as yourself**, not as `ec2-user`:

- It creates a Linux user on the instance **matching your local username**, with sudo, and installs your SSH **public** key into that user's `~/.ssh/authorized_keys`.
- For the keypair, spawn **uses your existing default SSH key** — `~/.ssh/id_ed25519` if present, else `~/.ssh/id_rsa` — and imports its public key to EC2. If you have no default key, spawn generates and manages its own under `~/.spawn/keys/`. (Windows targets require RSA, since the EC2 Administrator password can only be decrypted with an RSA key; if your default key isn't RSA, spawn falls back to a managed RSA key for those.)

So `spawn connect <name>` just works — it logs in as your user with the matching key. Nothing to configure.

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| `Unable to locate credentials` | Run `aws login` (or set a profile/keys); verify with `aws sts get-caller-identity`. |
| `AccessDenied` / `UnauthorizedOperation` on launch | Your IAM identity is missing an action — attach the [minimal policy](/reference/iam-permissions). |
| Acting in the wrong account | Set `SPORE_ACCOUNT` (or `--account`) to the intended account ID; check `AWS_PROFILE`/`SPORE_PROFILE`. |
| Credentials expired mid-session | `aws login` again (short-lived credentials refresh on re-login). |
