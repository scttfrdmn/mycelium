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
# From the official registry, by name
spawn plugin install tailscale --instance my-job --config auth_key=tskey-auth-...

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
      auth_key: tskey-auth-...
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
