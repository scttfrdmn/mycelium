# MPI Clusters

For workloads that need to communicate across multiple nodes — large-scale simulations, distributed training, or parallel data processing — spore.host can launch a multi-node MPI cluster as a single command.

## Launch a cluster

```sh
spawn launch \
  --name climate-sim \
  --instance-type hpc7g.16xlarge \
  --count 8 \
  --mpi \
  --ttl 12h \
  --command "mpirun -n 128 ./simulate --config config.yaml"
```

`--count 8` launches 8 instances. `--mpi` sets up passwordless SSH between all nodes, installs OpenMPI if not present, and configures the hostfile. The `--command` runs on node 0 after all nodes are ready.

MPI clusters launch **all-or-nothing**: spore.host waits for every node to come up and pass an MPI-readiness check (`mpirun` present, and the EFA fabric when `--efa` is set), then distributes the peer list to all nodes before the cluster is considered ready. If any node can't be placed, the whole launch fails and every already-launched node is cleaned up — so you never pay for a half-formed cluster.

## Availability Zone fallback

A cluster placement group lives in a single Availability Zone, and capacity for large HPC/GPU instance types is often available in one AZ but not another. If the primary AZ is out of capacity, spore.host automatically moves the **entire cluster** to the next AZ in the region — as a unit, so all nodes stay co-located in one placement group. This happens transparently; you'll see it noted in the launch output. No configuration is needed. (Pinning an explicit `--placement-group` fixes the cluster to that group's AZ and disables fallback.)

## EFA for high-performance networking

For workloads that saturate standard networking, enable Elastic Fabric Adapter:

```sh
spawn launch \
  --name distributed-training \
  --instance-type p4d.24xlarge \
  --count 4 \
  --mpi \
  --efa \
  --ttl 24h \
  --command "mpirun -n 32 --mca btl ^tcp python train.py"
```

`--efa` requires an EFA-supported instance type (p4d, hpc7g, c5n, and others). spore.host automatically selects an EFA-enabled security group and places all instances in a cluster placement group for lowest latency.

## Instance placement

By default, MPI clusters are placed in a cluster placement group named `spawn-mpi-<job-array-name>-<az>` for minimum latency. spawn creates it on demand in whichever AZ the cluster lands in (so [AZ fallback](#availability-zone-fallback) can pick a zone with capacity), and cleans up the groups for any AZs it tried and moved on from. If you have an existing placement group:

```sh
spawn launch \
  --name sim \
  --count 16 \
  --mpi \
  --placement-group my-hpc-group
```

## Monitoring the cluster

```sh
spawn list --job-array climate-sim     # all nodes in the cluster
spawn status climate-sim-0             # status of the head node
spawn status climate-sim-1             # status of worker node 1
```

## Lifecycle

The cluster's TTL applies to all nodes. If the job completes before the TTL (using `--on-complete terminate`), all nodes terminate together. If a worker node fails, the head node gets a notification.

```sh
spawn launch \
  --name sim \
  --count 8 \
  --mpi \
  --ttl 12h \
  --on-complete terminate \
  --completion-file /tmp/SPAWN_COMPLETE \
  --command "mpirun -n 64 ./sim && touch /tmp/SPAWN_COMPLETE"
```

::: tip
Write the completion file only from the head node (rank 0). MPI programs run on all nodes, so guard the `touch` with `if [ $OMPI_COMM_WORLD_RANK -eq 0 ]; then touch /tmp/SPAWN_COMPLETE; fi`.
:::

## Shared storage with FSx Lustre

For large datasets that all cluster nodes need to read, attach an FSx Lustre filesystem:

```sh
# Create a new FSx filesystem backed by S3
spawn launch sim \
  --count 8 --mpi --efa \
  --instance-type hpc6a.48xlarge \
  --fsx-create \
  --fsx-s3-bucket my-data-bucket \
  --fsx-import-path s3://my-data-bucket/inputs/ \
  --fsx-export-path s3://my-data-bucket/outputs/ \
  --fsx-mount-point /fsx

# Or attach an existing filesystem
spawn launch sim \
  --count 8 --mpi --efa \
  --fsx-id fs-0abc1234 \
  --fsx-mount-point /fsx
```

FSx Lustre is mounted at `/fsx` on every node. The filesystem ID and mount point are written as instance tags (`spawn:fsx-id`, `spawn:fsx-mount-point`) so boot scripts can auto-mount without hardcoding the ID.

::: info FSx compatibility
spore.host creates FSx `PERSISTENT_2` filesystems (Lustre 2.15 server). This is compatible with the standard Amazon Linux 2023 Lustre client (`dnf install -y lustre-client`). The mount uses port 988 and dynamic ports 1018–1023 — spawn automatically opens these in the instance security group so no manual SG configuration is needed.
:::
