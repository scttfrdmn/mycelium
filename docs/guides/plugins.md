# Plugins

Plugins install and manage software on a running instance — connecting to a
private network, mounting data transfer tooling, running a dev server. A plugin
is a declarative `plugin.yaml` spec describing lifecycle steps (install,
configure, start, health-check, stop) that spawn runs on the instance.

## Available plugins

The official registry lives at
[`spore-host/spore-plugins`](https://github.com/spore-host/spore-plugins):

| Plugin | What it does |
|--------|-------------|
| `tailscale` | Connect the instance to your Tailscale private network |
| `rstudio-server` | Browser-based R development environment |
| `globus-personal-endpoint` | High-speed data transfer via Globus Connect Personal |
| `spore-sync` | Live bidirectional directory sync |

## Installing a plugin

Install onto a running instance with `spawn plugin install <ref>`:

```sh
# From the official registry, by name.
# tailscale mints a short-lived key from your OAuth client, so you pass the ACL
# tag as config and the OAuth client via env — not a raw auth key.
export TS_API_CLIENT_ID=...  TS_API_CLIENT_SECRET=...
spawn plugin install tailscale --instance my-job --config tag=tag:spore

# Pin to a specific version
spawn plugin install rstudio-server@v1.0.0 --instance my-job

# From any GitHub repo
spawn plugin install github:myorg/my-plugins/my-tool --instance my-job

# From a local file (development)
spawn plugin install ./my-plugin.yaml --instance my-job
```

Per-plugin configuration is passed with repeatable `--config key=value` pairs.

Manage installed plugins:

```sh
spawn plugin list --instance my-job        # what's installed
spawn plugin status tailscale --instance my-job
spawn plugin remove tailscale --instance my-job
```

## Installing at launch

Declare plugins to install during startup with `--plugin` (repeatable; takes a
`ref[@version]`):

```sh
spawn launch analysis --instance-type r6i.4xlarge --plugin rstudio-server --ttl 8h
```

For per-plugin config, use a launch config file's `plugins:` block:

```yaml
# launch.yaml
instance_type: r6i.4xlarge
ttl: 8h
plugins:
  - ref: tailscale
    config:
      tag: tag:spore   # OAuth client via TS_API_CLIENT_ID/SECRET env (see above)
```

```sh
spawn launch analysis --config launch.yaml
```

## Writing a plugin

A plugin is a `plugin.yaml` file declaring lifecycle steps. Minimal example:

```yaml
name: my-tool                # kebab-case, must match the directory name
version: v1.0.0              # semver
description: "Install and run my-tool"
author: you

config:
  api_key:
    type: string             # string | int | bool
    required: true
    description: "API key for my-tool"

conditions:
  remote:
    - type: platform         # command | platform
      os: linux

remote:                      # steps run on the instance
  install:                   # phases: install, configure, start, stop, health
    - type: run              # remote step types: run | fetch | extract
      run: curl -fsSL https://example.com/install.sh | sh
  start:
    - type: run
      run: my-tool serve --key={{ config.api_key }}
  health:
    interval: 30s
    steps:
      - type: run
        run: my-tool status

outputs:
  endpoint:
    description: "Service endpoint"
```

Template references in the `config`, `instance`, `outputs`, and `pushed`
namespaces (for example `config.api_key` or `instance.name`, written in double
braces) are substituted at run time. See
[AUTHORING.md](https://github.com/spore-host/spore-plugins/blob/main/AUTHORING.md)
for the full spec, including controller-side `local` steps and the `push` API for
moving captured values to the instance.

### `plugin.yaml` field reference

**Top level:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Plugin id, kebab-case; must match the directory name. |
| `version` | string | SemVer (e.g. `v1.0.0`). |
| `description` | string | One-line summary. |
| `config` | map | User-supplied parameters, keyed by name (see below). |
| `conditions` | block | `local` / `remote` lists of preconditions checked before running. |
| `local` | block | Steps that run on **your machine** (the controller). |
| `remote` | block | Steps that run on **the instance**. |
| `outputs` | map | Values surfaced after provisioning, keyed by name. |

**`config.<name>` (a parameter):**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `string`, `int`, or `bool`. |
| `required` | bool | Fail if the user didn't supply it. |
| `default` | any | Value used when unset. |
| `description` | string | Shown in help/validation. |

**`conditions.local[]` / `conditions.remote[]`:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `command` (a probe command must succeed) or `platform`. |
| `run` | string | Command to run for `type: command`. |
| `os` | string | Required OS for `type: platform` (e.g. `linux`). |
| `message` | string | Shown when the condition fails. |

**`remote` phases:** `install`, `configure`, `start`, `stop`, each a list of
[steps](#step-fields); plus `health` (`interval` + `steps`) for the recurring
health-check loop.

**`local` block:** `provision`, `deprovision`, and `reconcile` (re-run when the
instance's IP changes after a stop/start) step lists, plus `env_passthrough` — the
allowlist of controller env vars a local step may read. Local steps otherwise run
with a minimal environment (`PATH`+`HOME` only) so plugin scripts can't scoop up
your AWS/other credentials; a plugin that needs a controller-side secret (e.g.
Tailscale's `TS_API_CLIENT_SECRET`) lists it here and spawn injects only those.

#### Step fields

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | `run`, `fetch`, `extract`, or `push`. |
| `run` | string | Shell command (`type: run`). |
| `url` / `dest` | string | Download source / destination (`type: fetch`). |
| `src` / `dest` | string | Archive path / target dir (`type: extract`). |
| `key` / `value` | string | Value to push to the instance (`type: push`). |
| `background` | bool | Run without waiting (e.g. a long-lived server). |
| `capture` | map | `varname` → JMESPath into the step's stdout JSON, for later template use. |
| `env` | map | Extra environment for this step. |
| `as_user` | bool | Run a remote `run` step as the instance's login user, not root (for tools that refuse root, e.g. Globus Connect Personal). |

**`outputs.<name>.source`** is `local_capture` (captured by a `local` step) or
`pushed` (delivered via the push API).

### Validate before you ship

Lint a spec offline (no instance, no AWS) with `spawn plugin validate`:

```sh
spawn plugin validate ./my-tool/plugin.yaml
spawn plugin validate plugins/*/plugin.yaml      # whole registry
```

It checks schema, semver, that the directory matches the plugin name, that step
and condition types are valid for their context, and that every config template
reference points at a declared parameter. The official registry runs this in CI
on every change.

## Contributing to the registry

Open a PR against [`spore-host/spore-plugins`](https://github.com/spore-host/spore-plugins)
adding `plugins/<name>/plugin.yaml`. CI validates it automatically; gated
integration tests then install it on a real instance.

## Data movement patterns

A common companion to plugins is moving data on and off the instance around your
job. The `--pre-stop` hook syncs results out before any shutdown — TTL expiry,
idle stop, or Spot interruption:

```sh
spawn launch process --instance-type r7i.4xlarge --ttl 8h \
  --pre-stop "aws s3 sync /data/output s3://my-bucket/output/" \
  --command "python process.py --input /data/input --output /data/output"
```

For persistent shared storage across instances, mount EFS:

```sh
spawn launch analysis --efs-id fs-0abc123 --efs-mount /shared \
  --command "python analyze.py --data /shared/datasets --output /shared/results"
```

Data written to `/shared` persists after the instance terminates.
