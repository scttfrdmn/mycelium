---
description: "Beyond launching and connecting, spawn has a set of operational commands for moving large data onto instances, keeping the spored agent current, and finding…"
---

# Managing instances & data

Beyond launching and connecting, spawn has a set of operational commands for
moving large data onto instances, keeping the `spored` agent current, and finding
what's running (or lingering) in your account. This guide covers those; every flag
is in the [spawn command reference](/tools/reference/spawn).

## Staging data across regions

Downloading the same dataset into every instance of a multi-region sweep pays the
cross-region transfer rate ($0.09/GB) over and over. `spawn stage` replicates the
data **once** into a regional S3 bucket per region, so each instance downloads from
its own region for free.

```sh
spawn stage estimate ./reference-db --regions us-east-1,us-west-2  # what you'd save
spawn stage upload ./reference-db --regions us-east-1,us-west-2 \
  --dest /mnt/data/reference-db
spawn stage list                 # what's staged, where
spawn stage delete <id>          # remove staged data when the sweep is done
```

Staged data lands at `--dest` on each instance (default `/mnt/data/<filename>`).
Associate an upload with a sweep via `--sweep-id` to track it alongside the run.

## Large reference data as an attached volume

For read-only reference data that's too big to bake into an AMI — a Kraken2 DB, a
BLAST index, ML weights — build an **EBS snapshot** once and attach it at launch,
instead of re-downloading it on every instance.

```sh
# Build a snapshot from a directory, tarball, or raw image (no instance launched)
spawn snapshot create --from ./kraken2-db --size 200 \
  --name kraken2-standard --description "Kraken2 standard DB"

# Attach it read-only at launch (repeatable; :ro is the common case)
spawn launch bio-run --attach-volume snap-0abc123:/mnt/kraken2:ro
```

`spawn snapshot create` accepts a local path or `s3://…` source and can encrypt
with a `--kms-key`. Inside a running instance, `spawn snapshot mount` creates and
mounts a volume from a snapshot on the spot. Because the data lives on a snapshot,
many instances can attach the same reference set without duplicating it into
custom AMIs.

## Keeping spored current

`spored` is the in-instance lifecycle agent that enforces the TTL and runs the
completion/idle/pre-stop hooks. To move a long-running instance onto a newer agent
**without** terminating it — and without losing its lifecycle state (the TTL
deadline, accumulated compute-seconds, and hook config all live in EC2 tags the new
agent re-reads) — use `upgrade-spored`:

```sh
spawn upgrade-spored <instance-id>              # to the latest release
spawn upgrade-spored <instance-id> --version 0.75.0
```

The swap is driven over SSM. A downgrade is refused unless you pass `--force`.

## Seeing what's running — and what's orphaned

spawn tags every resource it creates with `spawn:managed=true`, so it can inventory
them via the Resource Groups Tagging API:

```sh
spawn resources                  # everything spore.host created (yours) in the region
spawn resources --all-regions    # across every enabled region
spawn resources --all            # include resources other principals created
```

`spawn orphans` narrows that to resources that look **abandoned** — and are still
billing:

- EBS volumes in the `available` state (detached)
- security groups attached to no instance
- Elastic IPs that are unassociated or attached to a stopped instance (an EIP bills
  even while the instance is stopped)
- the shared key pair / IAM role when no instances remain

```sh
spawn orphans                    # your orphaned resources in this region
spawn orphans --all-regions
```

Run `spawn orphans` after a batch of work to catch anything the lifecycle reaper
didn't — leftover volumes and unassociated EIPs are the usual small, silent costs.

::: tip Cost hygiene
Instances always carry a TTL and terminate themselves, but volumes, EIPs, and
security groups can outlive them. `spawn orphans` is the fast way to find those.
:::
