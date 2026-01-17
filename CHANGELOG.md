# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Sweep management command (Issue #27)
  - `spawn list-sweeps` command to list all parameter sweeps from DynamoDB
  - Filter by status: `--status RUNNING`, `--status COMPLETED`, etc.
  - Filter by date: `--since 2026-01-15`
  - Limit results: `--last 10`
  - JSON output: `--json`
  - Table display with status icons, progress, and relative timestamps
  - Shows sweep ID, name, status, progress (launched/total), region, and creation time
- MPI (Message Passing Interface) support for distributed computing (Issue #28)
  - `spawn launch --mpi` flag to enable MPI cluster setup
  - `--mpi-processes-per-node` flag to control MPI process slots (defaults to vCPU count)
  - `--mpi-command` flag to specify command to run via mpirun
  - Automatic OpenMPI installation on all cluster nodes
  - Passwordless SSH setup between nodes using S3 for key distribution
  - MPI hostfile generation from peer discovery
  - Leader/worker coordination (index 0 runs mpirun)
  - S3 lifecycle policy for automatic MPI SSH key cleanup (1 day expiration)
  - Works seamlessly with existing job array infrastructure
- MPI compatibility with AMI creation workflow (Issue #30)
  - `--skip-mpi-install` flag to skip MPI installation on custom AMIs
  - Auto-detection of pre-installed MPI (checks for mpirun in PATH)
  - MPI environment configuration always runs (even if MPI pre-installed)
  - Supports custom MPI implementations (Intel MPI, MPICH, custom OpenMPI builds)
  - SSH setup and hostfile generation work identically with base or custom AMIs
  - Enables fast cluster launch: pre-install MPI in AMI, save 2-3 minutes per cluster
- EFS (Elastic File System) mounting support (Issue #32-33, Milestones 1-2 of #29)
  - `spawn launch --efs-id` flag to mount existing EFS filesystem
  - `--efs-mount-point` flag to customize mount location (default: /efs)
  - `--efs-profile` flag for performance profiles: general, max-io, max-throughput, burst
  - `--efs-mount-options` flag for custom NFS mount options (overrides profile)
  - Performance profiles optimize for different workloads (many small files, large sequential I/O, burst patterns)
  - Automatic NFS client installation and fstab configuration
  - Works with single instances, job arrays, and MPI clusters
  - All instances in job array share the same EFS filesystem
- FSx Lustre creation with S3 backing (Issue #34, Milestone 3 of #29)
  - `spawn launch --fsx-create` flag to create new FSx Lustre filesystem with S3 backing
  - `--fsx-id` flag to mount existing FSx Lustre filesystem
  - `--fsx-recall` flag to recreate filesystem from previous S3-backed configuration
  - `--fsx-storage-capacity` flag to specify capacity in GB (1200, 2400, or increments of 2400)
  - `--fsx-s3-bucket` flag to specify S3 bucket for import/export
  - `--fsx-import-path` and `--fsx-export-path` flags for S3 paths
  - `--fsx-mount-point` flag to customize mount location (default: /fsx)
  - Automatic S3 bucket creation if not exists
  - Data Repository Association (DRA) for lazy import from S3
  - Automatic export of modified files back to S3
  - Automatic Lustre client installation and filesystem mounting
  - Works with single instances, job arrays, and MPI clusters
  - Tags for tracking S3-backed filesystems for recall workflow
- FSx recall workflow enhancements (Issue #35, Milestone 4 of #29)
  - Enhanced S3 bucket tagging with FSx configuration for recall after deletion
  - S3 buckets tagged with stack name, storage capacity, import/export paths
  - `GetFSxConfigFromS3Bucket` function retrieves configuration from S3 bucket tags
  - `RecallFSxFilesystem` now falls back to S3 bucket tags if filesystem deleted
  - Enables true ephemeral compute: create → populate → export → delete → recall from S3
  - Recall works even after FSx filesystem is completely deleted
  - No DynamoDB required - configuration persists in S3 bucket tags
- FSx management commands (Issue #36, Milestone 5 of #29)
  - `spawn fsx list` - List all spawn-managed FSx filesystems across regions
  - `spawn fsx info <fs-id>` - Show detailed filesystem information
  - `spawn fsx delete <fs-id>` - Delete filesystem with confirmation prompt
  - `--export-first` flag for delete command (with manual export instructions)
  - `--yes` flag to skip confirmation prompts
  - Displays filesystem status, capacity, S3 backing, and cost estimates
  - Auto-discovers filesystem region across multiple AWS regions
- Future considerations documented in issues:
  - Issue #30: MPI compatibility with AMI creation workflow
- Cost estimation for parameter sweeps (Issue #25)
  - Pre-launch cost breakdown display for detached sweeps
  - Shows estimated costs for EC2 compute, Lambda orchestration, and S3 storage
  - `--estimate-only` flag to show cost estimate without launching
  - `-y/--yes` flag to auto-approve and skip confirmation prompt
  - Confirmation prompt asks user to approve before launching
  - Estimated cost stored in DynamoDB sweep record
  - Pricing data for 60+ instance types across 8 AWS regions
  - Automatic cost calculation based on instance type, region, and TTL
  - Prevents surprise AWS bills by showing costs upfront
- Data locality warnings for EFS and FSx (Issue #38)
  - Automatic region detection for EFS and FSx filesystems
  - Warns when launching instances in different region than storage
  - Shows cross-region data transfer costs ($0.02-0.08/GB depending on regions)
  - Estimates latency penalty (50-150ms for cross-region access)
  - Provides recommendations to launch in same region as storage
  - `--skip-region-check` flag to bypass warnings
  - Works with `--yes` flag for automated workflows
  - Helps prevent unexpected data transfer charges

### Fixed
- Lambda orchestrator cancellation race condition (Issue #26)
  - Added `cancel_requested` flag to SweepRecord structure
  - Lambda polling loop now checks for cancellation request every iteration
  - Prevents Lambda from overwriting CANCELLED status back to RUNNING
  - Ensures sweep stops orchestration immediately when cancelled by user
  - Deployed to production (Account 966362334030, Lambda: spawn-sweep-orchestrator)

## [0.5.0] - 2026-01-16

### Added
- Detached mode for parameter sweeps with Lambda orchestration (Issue #21)
- `spawn status --sweep-id` command to monitor detached sweeps from any machine
- `spawn cancel --sweep-id` command to cancel running sweeps and terminate instances
- `spawn resume --sweep-id --detach` to resume sweeps in Lambda mode from checkpoint
- Parameter validation for sweeps (validates instance types exist in target regions)
- Enhanced `spawn list` command with sweep grouping and collapsible sections (Issue #20)
- Estimated completion time display in sweep status
- Failed launch tracking with error messages in DynamoDB
- S3 parameter storage for unlimited sweep file sizes
- DynamoDB state management for cross-machine monitoring
- Lambda self-reinvocation pattern for multi-hour sweeps (13min polling + reinvoke)
- Dashboard OAuth authentication (Google, GitHub placeholder, Globus Auth)
- Expandable instance details in dashboard UI
- Cross-account IAM role support for sweep orchestration
- Documentation: PARAMETER_SWEEPS.md and DETACHED_MODE.md

### Fixed
- Cross-account IAM role trust policy to use IAM role principal
- Always include spored EC2 permissions in custom IAM roles
- Upload script to use correct spored binary names
- Adaptive logo rendering in dashboard (light/dark mode)
- JavaScript errors in dashboard
- OAuth response_type for Globus Auth provider

### Changed
- Improved dashboard table layout and column widths
- Better responsive design for dashboard

## [0.4.0] - 2026-01-14

### Added
- AMI management commands (`spawn create-ami` and `spawn list-amis`) for reusable software stacks
- AMI health checks for base AMI age tracking
- IAM instance profile support for secure AWS service access
  - 13 built-in policy templates (S3, DynamoDB, SQS, ECR, Secrets Manager, etc.)
  - Custom policy support via `--iam-policy-file`
  - AWS managed policy support via `--iam-managed-policies`
  - Named role support with `--iam-role` for reusability
- Job array support for coordinated instance groups
  - Launch N instances coordinately with `--count` and `--job-array-name`
  - Peer discovery via `/etc/spawn/job-array-peers.json`
  - Environment variables: `$JOB_ARRAY_INDEX`, `$JOB_ARRAY_SIZE`, `$JOB_ARRAY_ID`, `$JOB_ARRAY_NAME`
  - Group DNS: `{name}.{account}.spore.host` resolves to all IPs
  - Batch operations: `spawn list --job-array-name`, `spawn extend --job-array-name`
- Account-level tagging to spawn CLI
- Spore.host landing page and web dashboard foundation
- Logo image files for light and dark modes with adaptive integration

### Fixed
- AMI health check warnings to show newer base AMI availability more clearly

## [0.3.0] - 2025-12-23

### Added
- Lambda DNS gateway for spore.host domain (serverless DNS updates)
- Completion signal feature for workload-driven instance lifecycle
  - `spored complete` command to signal work completion
  - `/tmp/SPAWN_COMPLETE` file detection (universal signaling)
  - Configurable actions: terminate, stop, or hibernate on completion
  - Configurable grace period (default 30s) via `--completion-delay`
  - Priority system: Spot interruption > Completion > TTL > Idle
- Internationalization (i18n) support for spawn and truffle CLIs
  - 6 languages: English, Spanish, French, German, Japanese, Portuguese
  - 443+ translation keys covering all CLI output
  - Language detection from `--lang` flag, `SPAWN_LANG` env, system locale
  - Accessibility mode (`--accessibility` flag) for screen readers
  - No-emoji mode (`--no-emoji` flag) for cleaner terminal output
- Instance management commands
  - `spawn list`: List all spawn-managed instances across regions
    - Filters: `--region`, `--state`, `--instance-type`, `--family`, `--tag`
    - Output formats: table (default), JSON (`--format json`), YAML (`--format yaml`)
    - Human-readable age format (2h30m, 5d6h)
  - `spawn extend`: Extend instance TTL to prevent termination
    - Flexible time formats: 30m, 2h, 1d, 3h30m, 1d2h30m
    - Works with instance ID or name
    - Updates spored configuration on running instances
  - `spawn connect` / `spawn ssh`: SSH connection (both commands are aliases)
    - Auto-resolves SSH keys from `~/.ssh/` directory
    - Fallback to AWS Session Manager if no public IP
    - Custom user, port, and key support
- Enhanced monitoring in spored
  - Disk I/O threshold detection (100KB/min)
  - GPU utilization monitoring with nvidia-smi
  - Multi-GPU support (reports max utilization across GPUs)
  - Partition detection for accurate disk stats
- Comprehensive test suite
  - 667+ test cases across 48+ test functions
  - i18n validation tests (443 keys, 6 languages)
  - Command tests (list, extend, connect)
  - Monitoring tests (disk I/O, GPU, idle detection)
  - Table-driven tests with edge case coverage
- Documentation
  - TESTING.md: Complete testing guide
  - spawn/MONITORING.md: Comprehensive monitoring documentation
  - Enhanced spawn/README.md with detailed command examples
  - DNSSEC_CONFIGURATION.md: DNS security setup

### Fixed
- I18n command description translation formatting issues
- Format string safety in wizard.go (non-constant format strings)
- Cross-account DNS API request support in Lambda
- Goreleaser config for spored (renamed from spawnd)

### Changed
- Renamed `spawnd` → `spored` for consistency
- Enhanced idle detection to include disk I/O and GPU metrics
- Improved SSH key resolution with multiple pattern matching

## [0.2.0] - 2025-12-21

### Added
- Essential instance management (Phase 1 features)
- Comprehensive feature roadmap documentation

## [0.1.2] - 2025-12-21

### Fixed
- Truffle dependency to use fully qualified package name

## [0.1.1] - 2025-12-21

### Added
- Truffle dependency to spawn package

### Fixed
- Goreleaser v2 deprecation warnings

## [0.1.0] - 2025-12-21

### Added - truffle
- **Search command**: Find instance types by pattern with fuzzy matching
- **Spot command**: Discover and compare Spot instance prices across regions
- **Capacity command**: Find ML capacity (Capacity Blocks, ODCRs, reservations)
- **Quotas command**: View and manage AWS Service Quotas
- `--check-quotas` flag for pre-launch quota validation
- Python bindings with native cgo (10-50x faster than boto3)
- Multi-region support for simultaneous queries
- JSON output for piping to spawn
- No-credential mode (most features work without AWS credentials)

### Added - spawn
- Interactive wizard with 6-step guided setup for beginners
- Pipe mode to accept JSON input from truffle
- Direct mode with command-line flags
- Windows 11/10 native support
- Platform detection for Windows/Linux/macOS paths
- SSH key management with auto-creation
- AMI auto-detection (4 variants: x86/ARM × GPU/non-GPU, AL2023)
- Live progress display with real-time step-by-step updates
- Cost estimates (hourly and total) before launch
- Auto-termination via TTL and idle monitoring
- Hibernation support (pause instead of terminate when idle)
- S3 distribution with regional buckets for fast binary downloads
- Multi-architecture support (x86_64 and ARM64 Graviton)
- GPU support with auto-selected GPU-enabled AMIs

### Added - spored (spawnd)
- TTL monitoring with auto-terminate after configured time limit
- Idle detection monitoring CPU and network activity
- Hibernation capability (can hibernate instead of terminate)
- Self-monitoring via instance tag configuration
- systemd integration as proper Linux daemon with auto-restart
- Laptop-independent operation
- Graceful warnings (5 minutes before action)

### Added - Documentation
- README.md for each component
- QUICK_REFERENCE.md - Command cheat sheet
- COMPLETE_ECOSYSTEM.md - Full ecosystem overview
- truffle/QUOTAS.md - Quota management guide
- spawn/ENHANCEMENTS.md - S3/Windows/Wizard details
- spawn/IMPLEMENTATION.md - Technical details

### Added - Build System
- Multi-platform builds (Linux x86_64/ARM64, macOS Intel/Apple Silicon, Windows x86_64/ARM64)
- Makefile for each component
- Top-level Makefile for entire suite
- Installation targets for easy deployment
- Goreleaser v2 configuration for automated releases
- GitHub token configuration for Homebrew and Scoop repositories

### Fixed
- Goreleaser v2 compatibility issues
- Scoop configuration to use "scoops" (plural)
- Archive structure to avoid directory conflicts

[unreleased]: https://github.com/scttfrdmn/mycelium/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/scttfrdmn/mycelium/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/scttfrdmn/mycelium/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/scttfrdmn/mycelium/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/scttfrdmn/mycelium/compare/v0.1.2...v0.2.0
[0.1.2]: https://github.com/scttfrdmn/mycelium/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/scttfrdmn/mycelium/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/scttfrdmn/mycelium/releases/tag/v0.1.0
