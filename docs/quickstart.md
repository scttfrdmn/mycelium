---
description: "This guide gets you from zero to a running EC2 instance in about five minutes."
---

# Quick Start

This guide gets you from zero to a running EC2 instance in about five minutes. You'll need an AWS account with credentials configured — everything else is handled for you.

## Install

::: code-group

```sh [macOS / Linux (Homebrew)]
brew install spore-host/tap/truffle
brew install spore-host/tap/spawn
```

```powershell [Windows (Scoop)]
scoop bucket add spore-host https://github.com/spore-host/scoop-bucket
scoop install truffle spawn
```

```sh [Manual]
# Download the latest release assets for your OS/arch, then extract onto PATH.
OS=$(uname -s); ARCH=$(uname -m)
for tool in spawn truffle; do
  url=$(curl -fsSL "https://api.github.com/repos/spore-host/${tool}/releases/latest" \
    | grep -o "https://[^\"]*_${OS}_${ARCH}.tar.gz" | head -1)
  curl -fsSL "$url" -o "${tool}.tar.gz"
  tar -xzf "${tool}.tar.gz" "$tool"
  sudo install "$tool" /usr/local/bin/
done
```

:::

Verify the installation:

```sh
truffle --version
spawn --version
```

## Authenticate with AWS

spore.host uses whatever AWS credentials your shell already has. The recommended way to get them is **`aws login`** (AWS CLI v2.32.0+), which signs you in and refreshes short-lived credentials automatically:

```sh
aws login                       # sign in (opens your browser)
aws sts get-caller-identity     # confirm you're authenticated
```

Other paths work too — pick whichever matches your organization:
- **IAM Identity Center (SSO):** `aws configure sso` then `aws sso login --profile <name>` (set `AWS_PROFILE` or `--profile`).
- **Static keys / CI / assumed roles:** any credential your shell already resolves (env vars, `~/.aws/credentials`, instance role) is used as-is.

`aws login` is preferred where available because the credentials are short-lived.

Your credentials need permission to launch and manage EC2 instances; see the [minimal IAM policy](/reference/iam-permissions). For the full picture — profiles, `SPORE_*` config, and how your auth relates to what spore.host does on your behalf — see [AWS Authentication](/guides/aws-auth).

## Find an instance

Before launching, use `truffle` to find what's available and compare prices:

```sh
# Find a small instance in your region
truffle find "t3 medium"

# Find GPU instances with Spot prices
truffle find "nvidia gpu" --regions us-east-1
```

You'll see a table of matching instance types with vCPUs, memory, GPU specs, on-demand price, and current Spot price. Pick a type that fits your workload and budget.

## Launch your first instance

The simplest launch uses the interactive wizard. Just run `spawn` with no arguments:

```sh
spawn
```

The wizard walks you through instance type, region, SSH key, and TTL (the duration after which the instance automatically terminates). It takes about two minutes.

For a non-interactive launch:

```sh
spawn launch \
  --name my-first-instance \
  --instance-type t3.medium \
  --ttl 4h
```

Once running, you'll see the instance ID, hostname, and how to connect:

```
✓ Instance i-0a1b2c3d4e5f running
✓ my-first-instance.abc123.spore.host
✓ Connect: spawn connect my-first-instance
✓ Auto-terminates in 4h
```

## Connect

```sh
spawn connect my-first-instance
```

`spawn connect` is the canonical way in — it resolves the instance, uses the
right key, and logs you in as **your own username** (spawn creates a Linux user
matching your local login and imports your existing public key —
`~/.ssh/id_ed25519` or `~/.ssh/id_rsa` — generating a managed key only if you have
neither). Raw `ssh <you>@<public-ip>` works too, but prefer `spawn connect` so you
don't have to track the address, key, or username yourself.

## Check status

```sh
spawn list
spawn status my-first-instance
```

## Extend the TTL

If you need more time before the instance terminates:

```sh
spawn extend my-first-instance 8h
```

## Clean up

When you're done, **terminate** — this deletes the instance and its EBS volume,
so nothing keeps billing:

```sh
spawn terminate my-first-instance
```

Prefer to pause and resume later? **Stop** instead — but note a stopped instance
still bills for its EBS storage until you terminate it:

```sh
spawn stop my-first-instance       # resumable with `spawn start`; EBS still bills
```

(You didn't have to remember this: the `--ttl` you set means the instance
self-terminates at its deadline regardless.)

## Next steps

- **[How It Works](/how-it-works)** — understand the full lifecycle and how the tools connect
- **[Security, credentials & data flow](/architecture)** — what runs where, and why spore.host never holds your keys
- **[Costs & safety guarantees](/safety)** — the auto-terminate promise, its limits, and estimating cost up front
- **[Common Workflows](/guides/)** — end-to-end stories for real research tasks
- **[GPU Training](/guides/gpu-training)** — launch a GPU instance for a training job
- **[spawn launch reference](/tools/reference/spawn)** — every flag explained
