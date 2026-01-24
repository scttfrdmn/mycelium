# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.12.0] - 2026-01-24

This release focuses on production observability, cost management, reliability, and workflow orchestration.

### Added - Monitoring & Alerting System (Feature #58)

#### Alert Types
- **Cost Threshold Alerts**: Notify when sweep exceeds budget limit
- **Long-Running Alerts**: Detect sweeps running longer than expected
- **Failure Alerts**: Immediate notification on sweep/instance failures
- **Completion Alerts**: Success notifications with summary metrics

#### Notification Channels
- **Slack**: Rich formatted messages with cost breakdowns and status
- **Email**: Via SNS topics with HTML formatting
- **SNS**: Direct integration with AWS SNS for custom subscribers
- **Webhook**: Generic HTTP POST for custom integrations

#### Alert Management Commands
- **`spawn alerts create`**: Create alert with thresholds and channels
- **`spawn alerts list`**: List active alerts with status
- **`spawn alerts update`**: Modify alert configuration
- **`spawn alerts delete`**: Remove alert
- **`spawn alerts history`**: View alert trigger history

#### Infrastructure
- **Lambda Handler**: `alert-handler` for EventBridge trigger processing
- **DynamoDB Tables**: `spawn-alerts` (config), `spawn-alert-history` (audit log)
- **EventBridge Rules**: Dynamic rule creation per alert
- **TTL Cleanup**: 90-day automatic cleanup of alert history

#### Implementation
- **pkg/alerts/**: Alert configuration, validation, notification logic
- **lambda/alert-handler/**: Serverless alert processor
- **cmd/alerts.go**: Alert management CLI commands

### Added - Cost Tracking & Budget Management (Feature #59)

#### Real-Time Cost Estimation
- **Pre-Launch Estimates**: Show estimated cost before launching sweeps
- **Instance Pricing**: Real-time pricing data from AWS Pricing API
- **Multi-Region Support**: Per-region cost breakdowns
- **Instance Type Analysis**: Cost by instance type in mixed sweeps

#### Budget Management
- **Budget Limits**: Set dollar limits on sweeps with `--budget` flag
- **Budget Enforcement**: Prevent launches when budget exceeded
- **Remaining Budget**: Track budget consumption in real-time
- **Budget Alerts**: Integration with alerting system (#58)

#### Cost Reporting
- **Sweep Status**: Cost data in `spawn status --sweep-id` output
- **Regional Breakdown**: Cost per region for multi-region sweeps
- **Instance Hours**: Track total instance-hours consumed
- **Success Rate Correlation**: Cost vs success rate analysis

#### Cost Optimization Features
- **Spot Savings Display**: Show actual savings vs on-demand
- **Cost Projections**: Estimate completion cost for running sweeps
- **Historical Tracking**: Cost trends over time

#### Implementation
- **pkg/cost/**: Cost calculation engine, pricing API integration
- **pkg/pricing/**: AWS Pricing API client with caching
- **DynamoDB Integration**: Store cost data in sweep status records

### Added - Advanced Retry Strategies (Feature #60)

#### Retry Backoff Strategies
- **Fixed Delay**: Constant delay between retries
- **Exponential Backoff**: 2^attempt × base_delay (e.g., 1s, 2s, 4s, 8s)
- **Exponential with Jitter**: Random jitter (0-100%) to prevent thundering herd

#### Retry Configuration
- **Max Attempts**: Configurable retry limit per job
- **Base Delay**: Initial delay before first retry
- **Max Delay**: Cap on exponential growth
- **Jitter Factor**: Randomization percentage (0.0-1.0)

#### Intelligent Retry Logic
- **Exit Code Filtering**: Only retry specific exit codes
- **Blacklist Exit Codes**: Never retry certain failures (e.g., invalid input)
- **Per-Job Configuration**: Different retry strategies per job
- **Retry Tracking**: Record attempt count and delays in status

#### Queue Integration
- **Template Support**: Retry config in queue templates
- **Status Display**: Show retry attempts in `spawn queue status`
- **Result Preservation**: Keep results from all attempts
- **Failure Analysis**: Track which jobs exhaust retries

#### Implementation
- **pkg/queue/retry.go**: Retry calculation engine
- **pkg/agent/queue_runner.go**: Retry execution logic
- **Tests**: Comprehensive retry strategy unit tests

### Added - Workflow Orchestration Integration (Feature #61)

This is a **major feature** enabling spawn to integrate seamlessly with popular workflow orchestration tools through CLI enhancements rather than custom plugins.

#### Core Integration Flags

**`--output-id <file>`** - Write sweep/instance IDs to file for scripting
```bash
spawn launch --params sweep.yaml --detach --output-id /tmp/sweep_id.txt
# File contains: sweep-20240124-abc123
```

**`--wait`** - Block until sweep completion with automatic polling
```bash
spawn launch --params sweep.yaml --detach --wait --wait-timeout 2h
# Polls every 30s until COMPLETED/FAILED/CANCELLED
```

**`--check-complete`** - Standardized exit codes for workflow branching
```bash
spawn status $SWEEP_ID --check-complete
# Exit codes: 0=complete, 1=failed, 2=running, 3=error
```

#### Workflow Tool Examples

**11 Complete Working Examples:**
1. **Apache Airflow** - Custom operator + traditional DAG + TaskFlow API
2. **Prefect** - Task-based flows with retries and caching
3. **Nextflow** - Process-based bioinformatics pipelines
4. **Snakemake** - Rule-based reproducible workflows
5. **AWS Step Functions** - Serverless state machines with Lambda
6. **Argo Workflows** - Kubernetes-native orchestration
7. **Common Workflow Language (CWL)** - Portable tool definitions
8. **Workflow Description Language (WDL)** - Genomics pipelines
9. **Dagster** - Asset-based data orchestration with lineage
10. **Luigi** - Spotify's batch processing with dependency resolution
11. **Temporal** - Durable execution for long-running workflows

Each example includes:
- Complete working code
- README with setup instructions
- Sample input files
- Both simple (`--wait`) and advanced (manual polling) patterns

#### Documentation

**WORKFLOW_INTEGRATION.md** (1,088 lines)
- Quick start patterns (synchronous, asynchronous, fire-and-forget)
- Detailed integration guides for all 11 tools
- Advanced patterns (parallel sweeps, conditional execution, error recovery)
- Docker usage instructions
- Exit codes reference
- Troubleshooting guide
- Best practices

**examples/workflows/** - 37 files of examples and documentation

#### Docker Distribution

**Dockerfile** - Multi-stage Alpine-based build
- Minimal image size with alpine:latest
- Includes AWS CLI, openssh-client, jq, curl
- Multi-architecture: linux/amd64, linux/arm64

**.github/workflows/docker-spawn.yml** - Automated CI/CD
- Builds on version tags and main branch
- Pushes to Docker Hub: `scttfrdmn/spawn:latest`
- Version tags: `scttfrdmn/spawn:v0.12.0`
- Cache optimization with GitHub Actions

#### Implementation
- **cmd/launch.go**: Added `--output-id`, `--wait`, `--wait-timeout` flags (+98 lines)
- **cmd/status.go**: Added `--check-complete` flag with exit codes (+25 lines)
- **cmd/launch_test.go**: Unit tests for `writeOutputID`
- **cmd/status_test.go**: Unit tests for exit code logic (new file)

#### Use Cases Unlocked
- Scheduled parameter sweeps via workflow schedulers
- Multi-stage data pipelines with spawn compute
- CI/CD integration for ML model training
- Bioinformatics workflows with spawn + Nextflow
- Cost-optimized batch processing with retries

### Changed
- Exit codes for `spawn status --check-complete` are now standardized (0/1/2/3)
- `--wait` flag requires `--detach` (validation added)

### Documentation
- Added WORKFLOW_INTEGRATION.md (comprehensive 1,088-line guide)
- Added 11 workflow tool examples with READMEs
- Updated alerting documentation
- Updated cost tracking examples
- Updated retry strategy documentation

### Testing
- Added unit tests for workflow integration flags
- Added unit tests for retry strategies
- Added integration tests for alert system
- Added cost calculation tests

## [0.11.0] - 2026-01-24

### Added - Queue Templates (Feature #57)

#### Pre-built Templates
- **5 Production Templates**: Ready-to-use queue configurations for common workflows
  - `ml-pipeline` - ML training workflow (preprocess → train → evaluate → export)
  - `etl` - ETL pipeline (extract → transform → load → validate)
  - `ci-cd` - CI/CD workflow (checkout → build → test → deploy → smoke-test)
  - `data-processing` - Data processing (download → process → aggregate → upload)
  - `simple-sequential` - Simple 3-step customizable workflow
- **Variable Substitution**: `{{VAR}}` for required variables, `{{VAR:default}}` for optional
- **Embedded Templates**: Templates compiled into binary using go:embed for portability

#### Template Management Commands
- **`spawn queue template list`**: List all available templates with metadata
- **`spawn queue template show <name>`**: Display template details, jobs, and variables
- **`spawn queue template generate <name>`**: Generate queue config from template
  - `--var KEY=VALUE` flag for variable substitution
  - `--output <file>` flag to save generated config
  - Validates all required variables provided
  - Validates generated config before output

#### Interactive Wizard
- **`spawn queue template init`**: Interactive wizard to create custom queue configs
  - Guided prompts for queue metadata, jobs, dependencies, timeouts
  - Environment variable configuration per job
  - Retry strategy setup (max attempts, backoff: exponential/fixed)
  - Result path collection with glob pattern support
  - Global settings (timeout, failure handling, S3 bucket)
  - Saves validated config to `queue.json`

#### Custom Templates
- **User Template Directory**: `~/.config/spawn/templates/queue/`
- **Template Search Priority**: User config → embedded → filesystem
- **Override Built-ins**: User templates override embedded templates with same name
- Custom templates use same variable substitution and validation

#### Launch Integration
- **`spawn launch --queue-template <name>`**: Launch directly from template
- **`--template-var KEY=VALUE`**: Provide template variables inline
- Generates queue config on-the-fly without intermediate file
- Full validation before instance launch

#### Implementation
- **pkg/queue/template.go**: Template engine with variable substitution
- **pkg/queue/embedded.go**: Embedded template file system
- **pkg/queue/templates/**: 5 JSON templates embedded in binary
- **cmd/queue.go**: Template subcommands implementation
- **cmd/launch.go**: Launch integration with `--queue-template` flag
- **pkg/queue/template_test.go**: Comprehensive unit tests

#### Documentation
- **[BATCH_QUEUE_GUIDE.md](BATCH_QUEUE_GUIDE.md)**: Added "Queue Templates" section
  - Template listing and discovery
  - Variable substitution examples
  - Direct launch from templates
  - Custom template creation guide
  - All 5 template usage examples

### Testing
- **pkg/queue/template_test.go**: Template loading, variable extraction, substitution, validation
- Coverage: Variable parsing, defaults, missing required vars, template listing

## [0.10.0] - 2026-01-23

### Added - Scheduled Executions (Feature #51)

#### EventBridge Scheduler Integration
- **New Commands**: `spawn schedule create`, `spawn schedule list`, `spawn schedule describe`, `spawn schedule pause`, `spawn schedule resume`, `spawn schedule cancel`
- Schedule parameter sweeps for future execution without keeping CLI running
- One-time schedules with `--at` flag (ISO 8601 format)
- Recurring schedules with `--cron` flag (Unix cron expressions)
- Full timezone support via `--timezone` flag (IANA timezone database)
- Execution limits via `--max-executions` and `--end-after` flags
- Automatic sweep execution tracking in DynamoDB execution history

#### Infrastructure
- **DynamoDB Tables**: `spawn-schedules` and `spawn-schedule-history` with TTL
- **Lambda Function**: `scheduler-handler` for EventBridge trigger processing
- **S3 Buckets**: `spawn-schedules-{region}` for parameter file storage
- **EventBridge Scheduler**: Dynamic schedule creation per user request
- Cross-account IAM: Lambda in mycelium-infra → EC2 in mycelium-dev

#### Features
- Parameter file uploaded to S3 once, reused for each execution
- Pause/resume schedules without losing configuration
- Execution history with success/failure tracking
- Automatic cleanup after 90 days (DynamoDB TTL)
- Integration with existing sweep-orchestrator Lambda
- Full traceability: schedules linked to sweep executions

#### Documentation
- **[SCHEDULED_EXECUTIONS_GUIDE.md](SCHEDULED_EXECUTIONS_GUIDE.md)**: Comprehensive 800+ line guide
- Cron expression syntax and examples
- Timezone handling and DST transitions
- Best practices for scheduling strategies
- Troubleshooting common issues

### Added - Batch Queue Mode (Feature #52)

#### Sequential Job Execution
- **New Flag**: `spawn launch --batch-queue <file.json>` for sequential job pipelines
- **New Commands**: `spawn queue status <instance-id>`, `spawn queue results <queue-id>`
- Sequential job execution with dependency management
- Job-level retry with exponential or fixed backoff
- Global and per-job timeout enforcement
- Environment variable injection per job

#### Queue Features
- **Dependency Resolution**: Topological sort (Kahn's algorithm) for DAG validation
- **State Persistence**: Queue state saved to disk for crash recovery
- **Resume Capability**: Automatic resume from checkpoint after instance restart
- **Result Collection**: Incremental S3 upload of job outputs and logs
- **Failure Handling**: Configurable actions (`stop` or `continue`) on job failure
- **Result Paths**: Glob pattern support for collecting output files

#### Spored Integration
- New `spored run-queue` subcommand for queue execution
- Atomic state file writes (temp + rename) for crash safety
- Per-job stdout/stderr logging to `/var/log/spored/jobs/`
- Signal handling (SIGTERM, SIGINT) for graceful shutdown
- S3 upload of final queue state

#### Documentation
- **[BATCH_QUEUE_GUIDE.md](BATCH_QUEUE_GUIDE.md)**: Comprehensive 1,000+ line guide
- Complete JSON schema reference
- Dependency management patterns
- Retry strategy configuration
- ML pipeline examples (preprocess → train → evaluate → export)
- Troubleshooting queue execution issues

#### Examples
- **[ml-pipeline-queue.json](examples/ml-pipeline-queue.json)**: Production ML pipeline
- **[simple-queue.json](examples/simple-queue.json)**: Basic 3-step pipeline
- **[schedule-params.yaml](examples/schedule-params.yaml)**: Scheduling example with 11 configs
- **[simple-params.yaml](examples/simple-params.yaml)**: Simple 3-config sweep

### Added - Combined Features

#### Scheduled Batch Queues
- Schedule sequential job pipelines for recurring execution
- Example: Nightly ML training pipeline with preprocessing steps
- Full integration: EventBridge → Lambda → EC2 batch queue
- Execution history tracking for both schedules and queues

### Changed

#### Launch Command
- Added `--batch-queue` flag for queue mode
- Queue validation before instance launch
- User-data generation for queue runner bootstrap
- Single instance launch (no multi-region for queues)

#### Sweep Orchestrator
- Added `source` and `schedule_id` fields to sweep records
- Support for scheduler-initiated sweeps
- Backward compatible with CLI-initiated sweeps

#### Data Staging
- Added `UploadScheduleParams()` method for schedule parameter uploads
- Reuses existing multipart upload infrastructure

### Fixed
- Deprecated `io/ioutil` usage replaced with `io` and `os` packages
- Unnecessary nil checks removed for slice operations
- Optimized loop performance with direct append operations
- EventBridge Scheduler API field names corrected

### Testing

#### Unit Tests
- **pkg/scheduler/scheduler_test.go**: Schedule CRUD, EventBridge integration (69.7% coverage)
- **pkg/queue/queue_test.go**: Queue validation, config parsing (78.4% coverage)
- **pkg/queue/dependency_test.go**: Topological sort, cycle detection (78.4% coverage)
- **pkg/agent/queue_runner_test.go**: Job execution, state management, retry logic

#### Test Coverage
- Scheduler package: 69.7%
- Queue package: 78.4%
- Integration tests pending deployment

## [0.9.0] - 2026-01-22

### Added - HPC Integration & Cloud Migration

#### Slurm Integration
- **New Commands**: `spawn slurm convert`, `spawn slurm estimate`, `spawn slurm submit`
- Convert existing Slurm batch scripts (`.sbatch`) to spawn parameter sweeps
- Support for common Slurm directives: `--array`, `--time`, `--mem`, `--cpus-per-task`, `--gres=gpu`, `--nodes`
- Custom `#SPAWN` directives for cloud-specific overrides
- Automatic instance type selection based on resource requirements
- Cost estimation and comparison with institutional HPC clusters
- Comprehensive Slurm integration guide ([SLURM_GUIDE.md](SLURM_GUIDE.md))

#### Data Staging
- **New Commands**: `spawn stage upload`, `spawn stage list`, `spawn stage estimate`, `spawn stage delete`
- Multi-region data staging with automatic replication
- 90-99% cost savings for multi-region data distribution
- SHA256 integrity verification
- 7-day automatic cleanup (configurable 1-90 days)
- DynamoDB metadata tracking
- Integration with parameter sweeps via `--stage-id` flag
- Comprehensive data staging guide ([DATA_STAGING_GUIDE.md](DATA_STAGING_GUIDE.md))

#### MPI Enhancements
- **Placement Groups**: Automatic creation and management for low-latency MPI communication
- **EFA Support**: Elastic Fabric Adapter for ultra-low latency (sub-microsecond)
- **Instance Validation**: Pre-flight checks for EFA and placement group compatibility
- MPI placement group guide additions to [MPI_GUIDE.md](MPI_GUIDE.md)

### Added - Testing Infrastructure (Issue #53)

- **AWS Mocking Framework**: Full EC2 and S3 mock clients for unit testing (`pkg/aws/mock/`)
- **Test Utilities**: Comprehensive helper package (`pkg/testutil/`) with 20+ functions
- **Test Fixtures**: Example data in `testdata/` for Slurm scripts and parameters
- **Unit Tests**:
  - `cmd/launch_test.go`: Launch validation tests (405 lines, 7 test suites)
  - `pkg/aws/client_test.go`: AWS client tests (478 lines)
  - `cmd/slurm_test.go`: Slurm conversion tests (392 lines)
  - `cmd/stage_test.go`: Data staging tests (383 lines)

### Added - Documentation (Issue #53)

- **[SLURM_GUIDE.md](SLURM_GUIDE.md)**: Comprehensive 1,100+ line guide covering:
  - Quick start and conversion examples
  - Complete Slurm directive mapping
  - GPU, MPI, and array job examples
  - Migration workflow and cost comparison

- **[DATA_STAGING_GUIDE.md](DATA_STAGING_GUIDE.md)**: Complete 880+ line guide covering:
  - Cost optimization strategies
  - Multi-region deployment patterns
  - Integration with parameter sweeps
  - Bioinformatics and ML examples

- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)**: Comprehensive 1,200+ line guide covering:
  - Common errors (quota, permissions, network)
  - Launch, Slurm, staging, and MPI issues
  - Diagnostic commands and debugging workflows

### Added - Multi-Region Features

- **Per-region max concurrent limits** (Issue #41)
  - New `--max-concurrent-per-region` flag for balanced regional capacity usage
  - Prevents any single region from dominating global concurrent limit
  - Works alongside global `--max-concurrent` limit
  - Example: `spawn launch --max-concurrent 20 --max-concurrent-per-region 8`
  - Lambda orchestrator enforces limits during multi-region distribution

- **Spot instance type flexibility with fallback** (Issue #40)
  - Support instance type patterns: `c5.large|c5.xlarge|m5.large`
  - Wildcard expansion: `c5.*` tries all c5 types smallest to largest
  - Automatic fallback on `InsufficientInstanceCapacity` errors
  - Tracks requested vs. actual instance types in DynamoDB
  - Works with both single-region and multi-region sweeps
  - Pattern examples:
    - `p5.48xlarge|g6.xlarge|t3.micro` - Try GPU first, fallback to cheap
    - `c5.*` - Try all c5 sizes from smallest to largest
    - `m5.large|m5.xlarge` - Simple size progression

- **Regional cost breakdown in status command** (Issue #42)
  - Shows per-region instance hours and estimated costs
  - Tracks both terminated and running instance costs
  - Accumulates costs as instances complete
  - Example output:
    ```
    Regional Breakdown:
      us-east-1: 2/2 launched, 0 active, 0 pending, 0 failed
                 Cost: $2.40 (120.0 instance-hours)
      us-west-2: 2/2 launched, 0 active, 0 pending, 0 failed
                 Cost: $1.95 (130.0 instance-hours)

    Total Estimated Cost: $4.35
    ```

- **Multi-region result collection** (Issue #43)
  - `spawn collect-results` automatically detects multi-region sweeps
  - Queries all regional S3 buckets concurrently
  - Optional `--regions` flag to filter specific regions
  - CSV output includes region column for each result
  - Example: `spawn collect-results --sweep-id <id> --regions us-east-1,us-west-2`

- **Integration testing for multi-region features** (Issue #44)
  - Comprehensive test suite covering all multi-region scenarios
  - Tests for per-region limits, fallback, cost tracking, and collection
  - Run with: `go test -v -tags=integration ./...`
  - Tests validate against live AWS resources
  - Includes automatic cleanup of test resources

- **Dashboard multi-region support** (Issue #45)
  - Regional breakdown table with per-region progress and costs
  - Instance type column shows actual vs. requested types
  - Region filter dropdown for instance list
  - Multi-region indicator in sweep list view
  - Real-time cost tracking per region

### Changed

- Sweep launch now detects single-region parameters and uses correct region
  - Previously: single-region parameters used auto-detected region
  - Now: uses the region specified in parameters
  - Fixes issue where single-region sweeps launched in wrong region

- Cross-account role assumption now uses explicit mycelium-dev profile
  - Previously: used default AWS profile for account ID lookup
  - Now: explicitly loads mycelium-dev profile (435415984226)
  - Ensures instances launch in correct account

- Lambda orchestrator fallback logic extended to single-region sweeps
  - Previously: only multi-region sweeps had fallback logic
  - Now: both single-region and multi-region sweeps support patterns
  - Consistent behavior across all sweep types

### Fixed

- Parameter sweep mode no longer enters interactive wizard
  - Fixed by moving parameter sweep check before wizard/config logic in launch.go:256-264
  - Resolves integration test failures with `--param-file` flag

- Status command JSON output now properly marshals all fields
  - Added `json` struct tags to SweepRecord, RegionProgress, and SweepInstance
  - Enables `--json` flag for programmatic status queries
  - Required for integration tests and API consumption

- Instance type patterns no longer passed directly to EC2 API
  - Lambda orchestrator now parses patterns before RunInstances calls
  - Fixes `InvalidParameterValue` errors with pipe-separated types
  - Properly handles wildcards and fallback sequences

- Regional breakdown shows correct costs and instance hours
  - Fixed accumulation logic in Lambda orchestrator
  - Properly tracks terminated vs. running instance costs
  - Updates DynamoDB with accurate regional statistics

## [0.8.0] - 2026-01-16

### Added

- Multi-region parameter sweep support
- Detached sweep orchestration via Lambda
- Auto-detection of closest AWS region
- Distribution modes: fair-share and opportunistic
- Real-time sweep status with regional progress
- Sweep cancellation with cross-region cleanup

## Earlier Versions

See git history for changes prior to v0.8.0.

[0.11.0]: https://github.com/scttfrdmn/mycelium/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/scttfrdmn/mycelium/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/scttfrdmn/mycelium/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/scttfrdmn/mycelium/compare/v0.7.0...v0.8.0
