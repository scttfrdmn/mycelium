# spawnd Monitoring

The spawnd agent monitors instances and can auto-terminate when idle.

## Metrics Monitored

The spawnd agent monitors the following metrics to determine if an instance is idle:

1. **CPU Usage** - Threshold: <5% (configurable via `spawn:idle-cpu` tag)
2. **Network Traffic** - Threshold: ≤10KB/min
3. **Disk I/O** - Threshold: ≤100KB/min
4. **GPU Utilization** - Threshold: ≤5% (if nvidia-smi available)
5. **Active Terminals** - Checks `/dev/pts/` for active PTYs
6. **Logged-in Users** - Uses `who` command
7. **Recent User Activity** - Checks `wtmp` logs (last 5 minutes)

## Idle Detection

The system is considered **idle** when **ALL** conditions are met:
- CPU usage < threshold
- Network traffic ≤ threshold
- Disk I/O ≤ threshold
- GPU utilization ≤ threshold (if GPU present)
- No active terminals
- No logged-in users
- No recent user activity

If any single condition fails, the system is considered active.

## Configuration

Set monitoring behavior via EC2 instance tags when launching:

```bash
spawn launch --instance-type g5.xlarge \
  --idle-timeout 30m \               # Idle timeout duration
  --idle-cpu 5.0 \                   # CPU threshold (%)
  --hibernate-on-idle                # Hibernate instead of terminate
```

### Available Tags

- `spawn:idle-timeout` - Duration before terminating idle instance (e.g., "30m", "2h", "1d")
- `spawn:idle-cpu` - CPU usage percentage threshold (default: 5.0)
- `spawn:hibernate-on-idle` - Set to "true" to hibernate instead of terminate
- `spawn:ttl` - Maximum time-to-live regardless of activity
- `spawn:completion-tag` - Stop when specific tag appears

## Disk I/O Monitoring

The spawnd agent monitors disk I/O by reading `/proc/diskstats` and tracking:

### Monitored Device Types

- **xvd\*** - Xen virtual disks (e.g., xvda, xvdb)
- **nvme\*** - NVMe SSDs (e.g., nvme0n1, nvme1n1)
- **sd\*** - SCSI/SATA drives (e.g., sda, sdb)
- **vd\*** - virtio disks (e.g., vda, vdb)

### Partition Handling

The agent attempts to skip partitions to avoid double-counting:
- Partitions are identified by checking if device name ends with a digit and length > 4
- Examples: xvda1, nvme0n1p1, sdb2 are partitions
- Main devices: xvda, nvme0n1, sda are main block devices

**Note**: The current implementation has limitations with certain device naming schemes (e.g., nvme0n1 is incorrectly classified as a partition).

### Calculation

```
Total I/O = (sectors_read + sectors_written) × 512 bytes
Idle if: Total I/O ≤ 100KB per minute
```

## GPU Monitoring

GPU monitoring requires the `nvidia-smi` command-line tool.

### Detection

The agent checks for nvidia-smi availability at startup:
- If found: Monitors GPU utilization
- If not found: GPU monitoring returns 0% (system can still be idle based on other metrics)

### Query Method

```bash
nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits
```

### Multi-GPU Support

For systems with multiple GPUs:
- The agent queries all GPUs
- Returns the **maximum utilization** across all GPUs
- Example: GPU0=10%, GPU1=75%, GPU2=5% → Reports 75%

### Idle Threshold

GPU is considered idle when utilization ≤ 5%

## Debugging

Check if monitoring is working:

```bash
# SSH into instance
ssh into-instance

# Check spored status (if status command implemented)
sudo spored status
```

### Manual Metric Check

You can manually check metrics using the same methods spawnd uses:

```bash
# Check disk I/O
cat /proc/diskstats | awk '$3 ~ /^(xvd|nvme|sd|vd)/ {print $3, $6, $10}'

# Check GPU utilization (if nvidia-smi available)
nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits

# Check CPU usage
top -bn1 | grep "Cpu(s)"

# Check network traffic
cat /proc/net/dev

# Check logged-in users
who

# Check active terminals
ls /dev/pts/
```

## Use Cases

### Development Instances

Launch an instance that terminates after 30 minutes of inactivity:

```bash
spawn launch --instance-type m7i.large \
  --idle-timeout 30m \
  --name dev-instance
```

### Long-Running Tasks

Launch an instance for a long-running task with high TTL but idle detection:

```bash
spawn launch --instance-type c6a.xlarge \
  --ttl 24h \
  --idle-timeout 1h \
  --name batch-processing
```

The instance will terminate after either:
- 24 hours total (TTL)
- 1 hour of continuous idle time

### GPU Workloads

Launch a GPU instance that hibernates when idle:

```bash
spawn launch --instance-type g5.xlarge \
  --idle-timeout 15m \
  --hibernate-on-idle \
  --name ml-training
```

Hibernation preserves memory state and allows resuming work later.

### Cost-Conscious Development

Launch with aggressive idle detection to minimize costs:

```bash
spawn launch --instance-type t3.medium \
  --idle-timeout 10m \
  --idle-cpu 3.0 \
  --name quick-test
```

This terminates quickly after work is complete.

## Thresholds Explained

### Why These Thresholds?

- **CPU < 5%**: Allows for background system processes
- **Network ≤ 10KB/min**: Filters out periodic health checks and metrics
- **Disk I/O ≤ 100KB/min**: Allows for log rotation and system writes
- **GPU ≤ 5%**: Accounts for desktop compositing and minimal rendering

### Adjusting Thresholds

Currently, only CPU threshold is configurable via `--idle-cpu` flag:

```bash
spawn launch --instance-type m7i.large \
  --idle-timeout 30m \
  --idle-cpu 10.0  # More lenient CPU threshold
```

Network, disk, and GPU thresholds are hardcoded but could be made configurable in future versions.

## Monitoring Interval

The spawnd agent checks metrics every 60 seconds by default. This means:

- An instance must be **continuously idle** for the full idle-timeout duration
- Any activity resets the idle timer
- Short bursts of activity will prevent termination

## Limitations

### Partition Detection

The current partition detection logic has edge cases:
- **nvme0n1** (main device) is incorrectly skipped because it ends with '1'
- **sda1** (partition) is incorrectly included because length=4 is not >4
- **vda1** (partition) is incorrectly included for the same reason

This may result in slightly inaccurate disk I/O measurements but shouldn't significantly affect idle detection.

### GPU Detection

- Only NVIDIA GPUs are supported (requires nvidia-smi)
- AMD and Intel GPUs are not currently monitored
- GPU monitoring gracefully degrades if nvidia-smi is not available

### False Positives

The agent may incorrectly detect activity in some cases:
- Background system updates
- Cron jobs
- Log rotation
- System monitoring tools

Adjust thresholds if false positives are common in your environment.

## Safety Features

### Active User Protection

The agent will **never** terminate an instance with:
- Logged-in users (via `who`)
- Active terminal sessions (PTYs in `/dev/pts/`)
- Recent user activity (wtmp within last 5 minutes)

This prevents accidentally terminating instances while users are working.

### Spot Instance Interruption

For Spot instances, spawnd monitors the EC2 metadata service for interruption warnings and:
1. Sends notifications to logged-in users
2. Runs cleanup tasks
3. Gracefully terminates before AWS forcibly stops the instance

### Graceful Shutdown

When terminating, spawnd:
1. Cleans up DNS records
2. Notifies users (if any)
3. Allows cleanup hooks to run
4. Performs graceful shutdown

## Implementation Details

### File Locations

- **Disk stats**: `/proc/diskstats`
- **CPU stats**: `/proc/stat`
- **Network stats**: `/proc/net/dev`
- **PTY devices**: `/dev/pts/*`
- **User logs**: `/var/run/utmp`, `/var/log/wtmp`

### Error Handling

If the agent cannot read a metric (e.g., /proc files unavailable):
- The metric is assumed to be 0 (idle)
- Monitoring continues with remaining metrics
- Errors are logged but don't crash the agent

This ensures robustness in various environments.

## Testing

The monitoring logic is tested in `spawn/pkg/agent/monitoring_test.go` with tests for:

- Disk I/O parsing and threshold detection
- GPU utilization parsing and multi-GPU handling
- Partition detection logic
- Device type recognition
- Sector-to-byte conversion
- Idle detection with multiple conditions
- Real-world usage scenarios

Run tests:

```bash
cd spawn/pkg/agent
go test -v -run TestGetDiskIO
go test -v -run TestGetGPU
go test -v -run TestIsIdle
```
