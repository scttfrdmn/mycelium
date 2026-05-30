# Python SDK

::: tip Two SDKs are available
**REST API client** (`sdk/python/`) — manages instances and queries truffle via the spore.host HTTP API. Covered on this page.

**Native CGO bindings** (`truffle/bindings/python/`) — calls the truffle Go library directly from Python, 10–50× faster. See [`bindings/python/`](https://github.com/spore-host/truffle/tree/main/bindings/python) in the truffle repo.
:::

The `spore-host` Python package lets you discover EC2 instances, check Spot prices, and manage running instances from Python scripts, Jupyter notebooks, and reactive notebooks like marimo.

The SDK talks to the spore.host REST API. AWS credentials are used for authentication — no separate login is required.

::: tip Not on PyPI yet
Until the package is published, install directly from the repository:
```sh
pip install "git+https://github.com/spore-host/spore-host.git#subdirectory=sdk/python"
```
:::

## Requirements

- Python 3.9+
- AWS credentials configured (`~/.aws/credentials`, environment variables, or EC2 instance metadata)
- `boto3`, `requests` (installed automatically)

Optional notebook extras:
```sh
pip install "spore-host[jupyter]"
```

## Quick start

```python
import spore

# Find instances by natural language query
results = spore.truffle.find("amd epyc genoa", region="us-east-1")
for r in results:
    print(r.instance_type, f"${r.on_demand_price:.4f}/hr")

# Check Spot prices
prices = spore.truffle.spot("c8a.2xlarge", region="us-east-1")
cheapest = min(prices, key=lambda p: p.spot_price)
print(f"Cheapest: {cheapest.instance_type} in {cheapest.availability_zone} @ ${cheapest.spot_price:.4f}/hr")

# List running instances
instances = spore.spawn.list()
for inst in instances:
    print(inst.name, inst.state, inst.ttl)

# Manage an instance
inst = spore.spawn.status("sim-run-42")
inst.extend("4h")          # push the TTL deadline forward
inst.stop()                # stop (preserves instance)
inst.wait("terminated")    # block until it terminates
```

## Authentication

The SDK uses your ambient AWS credentials — the same ones the CLI uses. No extra configuration needed:

```python
import spore

# Uses ~/.aws/credentials or environment variables automatically
client = spore.Client()

# Use a specific AWS profile
client = spore.Client(profile="my-research-account", region="us-east-1")

# Use a spore.host API key (for hosted deployments)
client = spore.Client(api_key="sk_...")

# Point at a self-hosted deployment
client = spore.Client(api_url="https://spore.internal.university.edu")
```

Module-level `spore.truffle` and `spore.spawn` use a default `Client()` with ambient credentials.

---

## `spore.truffle` — instance discovery

### `truffle.find(query, region=None, regions=None)`

Find instance types matching a natural language query.

```python
# Single region
results = spore.truffle.find("nvidia h100 8gpu", region="us-east-1")

# Multiple regions
results = spore.truffle.find("arm64 64gb", regions=["us-east-1", "us-west-2", "eu-west-1"])

for r in results:
    print(r.instance_type, r.vcpus, "vCPU", r.memory_gib, "GiB")
    if r.gpus:
        print(f"  {r.gpus}× {r.gpu_model}")
```

**Returns:** `list[InstanceType]`

### `truffle.spot(instance_type, region=None, regions=None)`

Get current Spot prices across regions and AZs.

```python
prices = spore.truffle.spot("g5.xlarge", region="us-east-1")
prices.sort(key=lambda p: p.spot_price)

for p in prices[:5]:
    print(f"{p.availability_zone}: ${p.spot_price:.4f}/hr  ({p.savings_pct:.0f}% off on-demand)")
```

**Returns:** `list[SpotPrice]`

### `truffle.quota(instance_type, region, spot=False)`

Check whether your account has quota to launch an instance type.

```python
q = spore.truffle.quota("p4d.24xlarge", region="us-east-1")
if not q.can_launch:
    print(f"Quota check failed: {q.message}")
```

**Returns:** `QuotaInfo`

---

## `spore.spawn` — instance management

### `spawn.list(state="running", region=None)`

List spawn-managed instances.

```python
# All running instances across all regions
instances = spore.spawn.list()

# Stopped instances in a specific region
stopped = spore.spawn.list(state="stopped", region="us-east-1")
```

**Returns:** `list[Instance]`

### `spawn.status(instance_id_or_name)`

Get current status for a single instance.

```python
inst = spore.spawn.status("sim-run-42")
print(inst.state, inst.public_ip, inst.ttl)
```

**Returns:** `Instance`

### `spawn.stop(instance_id_or_name, hibernate=False)`

Stop a running instance. Pass `hibernate=True` to hibernate instead.

```python
spore.spawn.stop("sim-run-42")
spore.spawn.stop("rstudio", hibernate=True)
```

### `spawn.start(instance_id_or_name)`

Start a stopped or hibernated instance.

```python
spore.spawn.start("sim-run-42")
```

### `spawn.extend(instance_id_or_name, duration)`

Extend an instance's TTL deadline.

```python
spore.spawn.extend("sim-run-42", "4h")
```

---

## Instance object

Methods on a retrieved `Instance` object — chainable, no need to repeat the ID:

```python
inst = spore.spawn.status("sim-run-42")

inst.extend("2h")            # extend TTL
inst.stop()                  # stop
inst.stop(hibernate=True)    # hibernate
inst.start()                 # restart
inst.terminate()             # permanently terminate
inst.refresh()               # update state from API

# Block until a state is reached
inst.wait("running")         # wait for running
inst.wait("terminated")      # wait for completion
inst.wait_running()          # shorthand
inst.wait_done()             # wait until terminated
```

The `wait()` method accepts an `on_status` callback for progress display:

```python
inst.wait(
    "terminated",
    poll_interval=30,
    timeout=86400,
    on_status=lambda i: print(f"{i.name}: {i.state}"),
)
```

### Rich display in notebooks

`Instance` objects render as formatted cards in Jupyter and marimo:

```python
inst = spore.spawn.status("sim-run-42")
inst  # displays formatted card with state, type, IP, TTL
```

---

## Data classes

### `InstanceType`

| Field | Type | Description |
|-------|------|-------------|
| `instance_type` | str | EC2 instance type (e.g. `c8a.2xlarge`) |
| `region` | str | AWS region |
| `vcpus` | int | vCPU count |
| `memory_gib` | float | Memory in GiB |
| `architecture` | str | `x86_64` or `arm64` |
| `on_demand_price` | float | On-demand price per hour (USD) |
| `gpus` | int | GPU count |
| `gpu_model` | str | GPU model name (e.g. `A10G`) |
| `available_azs` | list[str] | Availability zones where type is offered |

### `SpotPrice`

| Field | Type | Description |
|-------|------|-------------|
| `instance_type` | str | EC2 instance type |
| `region` | str | AWS region |
| `availability_zone` | str | AZ (e.g. `us-east-1a`) |
| `spot_price` | float | Current Spot price (USD/hr) |
| `on_demand_price` | float | On-demand price for comparison |
| `savings_pct` | float | Savings vs on-demand (0–100) |

### `QuotaInfo`

| Field | Type | Description |
|-------|------|-------------|
| `instance_type` | str | Instance type checked |
| `region` | str | Region checked |
| `vcpus` | int | vCPU quota for this family |
| `can_launch` | bool | `True` if quota allows a launch |
| `message` | str | Human-readable explanation |

### `Instance`

| Field | Type | Description |
|-------|------|-------------|
| `instance_id` | str | EC2 instance ID |
| `name` | str | Instance name |
| `instance_type` | str | EC2 instance type |
| `state` | str | `running`, `stopped`, `terminated`, etc. |
| `region` | str | AWS region |
| `public_ip` | str | Public IP address |
| `dns` | str | spore.host DNS name |
| `launch_time` | datetime | Launch timestamp |
| `ttl` | str | TTL setting |
| `idle_timeout` | str | Idle timeout setting |
