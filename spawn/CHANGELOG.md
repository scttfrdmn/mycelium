# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0] - 2026-01-17

### Added

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

[0.9.0]: https://github.com/scttfrdmn/mycelium/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/scttfrdmn/mycelium/compare/v0.7.0...v0.8.0
