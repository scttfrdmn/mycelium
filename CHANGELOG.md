# Changelog

All notable changes to mycelium will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2025-12-19

### Added - truffle

- **Search command**: Find instance types by pattern with fuzzy matching
- **Spot command**: Discover and compare Spot instance prices across regions
- **Capacity command**: Find ML capacity (Capacity Blocks, ODCRs, reservations)
- **Quotas command**: View and manage AWS Service Quotas
- **--check-quotas flag**: Pre-launch quota validation
- **Python bindings**: Native cgo bindings (10-50x faster than boto3)
- **Multi-region support**: Query across multiple AWS regions simultaneously
- **JSON output**: Clean JSON for piping to spawn
- **No-credential mode**: Most features work without AWS credentials

### Added - spawn

- **Interactive wizard**: 6-step guided setup for beginners
- **Pipe mode**: Accept JSON input from truffle
- **Direct mode**: Launch with command-line flags
- **Windows support**: Native support for Windows 11/10
- **Platform detection**: Auto-detects Windows/Linux/macOS paths
- **SSH key management**: Auto-creates keys if missing
- **AMI auto-detection**: 4 variants (x86/ARM, GPU/non-GPU, AL2023)
- **Live progress display**: Real-time step-by-step updates
- **Cost estimates**: Shows hourly and total costs before launch
- **Auto-termination**: TTL and idle monitoring
- **Hibernation support**: Pause instead of terminate when idle
- **S3 distribution**: Regional S3 buckets for fast spawnd downloads
- **Multi-architecture**: Supports x86_64 and ARM64 (Graviton)
- **GPU support**: Auto-selects GPU-enabled AMIs for P/G instances

### Added - spawnd

- **TTL monitoring**: Auto-terminate after configured time limit
- **Idle detection**: Monitors CPU and network activity
- **Hibernation**: Can hibernate instead of terminate
- **Self-monitoring**: Reads configuration from instance tags
- **systemd integration**: Proper Linux daemon with auto-restart
- **Laptop-independent**: Works even when user disconnects
- **Graceful warnings**: Warns user 5 minutes before action

### Documentation

- README.md for each component
- QUICK_REFERENCE.md - Command cheat sheet
- COMPLETE_ECOSYSTEM.md - Full ecosystem overview
- truffle/QUOTAS.md - Quota management guide
- spawn/ENHANCEMENTS.md - S3/Windows/Wizard details
- spawn/IMPLEMENTATION.md - Technical details

### Build System

- Multi-platform builds (Linux x86_64/ARM64, macOS Intel/M1, Windows)
- Makefile for each component
- Top-level Makefile for entire suite
- Installation targets

## [0.3.0] - 2025-12-23

### Added - Internationalization 🌍

- **6 Language Support**: English, Spanish, French, German, Japanese, Portuguese
- **443 Translation Keys**: Complete CLI translation coverage
- **Accessibility Mode**: Screen reader support with `--accessibility` flag
- **No-emoji Mode**: `--no-emoji` flag for cleaner output
- **Language Detection**: Auto-detects from `LANG`, `LC_ALL`, or `SPAWN_LANG` environment variables
- **Per-command Language**: Use `--lang` flag to override language for any command

### Added - Completion Signals ✅

- **Workload-Driven Lifecycle**: Instances can signal when work is complete
- **`spored complete` Command**: Signal completion from within instance
- **Auto-Actions**: Terminate, stop, or hibernate on completion
- **Grace Period**: Configurable delay before action (default 30s)
- **File-Based Signaling**: Universal `/tmp/SPAWN_COMPLETE` file support
- **Priority System**: Spot interruption > Completion > TTL > Idle

### Added - Instance Management Commands 📋

- **`spawn list`**: List all spawn-managed instances across regions
  - Filter by region, state, instance type/family, or tags
  - Multiple filter support with AND logic
  - JSON/YAML output for automation
  - Human-readable age format (2h30m, 5d6h)
- **`spawn extend`**: Extend instance TTL to prevent termination
  - Flexible time formats (30m, 2h, 1d, 3h30m)
  - Update running instances without restart
  - Works with instance ID or name
- **`spawn connect` / `spawn ssh`**: SSH alias support
  - Both commands work identically
  - Auto-resolves SSH keys from `~/.ssh/`
  - Fallback to AWS Session Manager
  - Custom user, port, and key support

### Added - Monitoring & Testing 🧪

- **Comprehensive Test Suite**: 667+ test cases across 48+ test functions
  - i18n validation tests (443 keys, 6 languages)
  - Command tests (list, extend, connect)
  - Monitoring tests (disk I/O, GPU, idle detection)
- **Enhanced Monitoring**:
  - Disk I/O threshold detection (100KB/min)
  - GPU utilization monitoring with nvidia-smi
  - Multi-GPU support (reports max utilization)
  - Partition detection for accurate disk stats
- **Test Infrastructure**:
  - Makefile test targets (`test`, `test-i18n`, `test-coverage`, `test-coverage-report`)
  - Table-driven tests with comprehensive edge cases
  - Mock data and temporary file handling

### Added - Documentation 📚

- **TESTING.md**: Complete testing guide
  - Running tests
  - Writing tests (patterns and examples)
  - Test organization
  - Coverage goals and CI/CD integration
- **spawn/MONITORING.md**: Comprehensive monitoring documentation
  - Metrics monitored (CPU, network, disk I/O, GPU)
  - Idle detection logic
  - Configuration via EC2 tags
  - Debugging and use cases
- **Enhanced spawn/README.md**:
  - Commands section with detailed examples
  - spawn list filtering examples
  - spawn extend TTL format rules
  - spawn connect/ssh key resolution

### Added - DNS & Infrastructure 🌐

- **Lambda DNS Gateway**: Serverless DNS updates for spore.host
- **Cross-Account DNS**: Support for DNS updates across AWS accounts
- **DNSSEC Support**: Enhanced DNS security configuration
- **Automatic DNS Registration**: Instances auto-register on startup

### Fixed 🐛

- **i18n Command Descriptions**: Fixed translation formatting issues
- **Format String Safety**: Fixed non-constant format string in wizard.go
- **Portuguese Support**: Added complete Portuguese translations
- **Cross-Account Requests**: Fixed Lambda permissions for cross-account DNS

### Changed 🔄

- **spawnd → spored**: Renamed daemon for consistency
- **Enhanced Idle Detection**: Now includes disk I/O and GPU metrics
- **Improved Key Resolution**: Better SSH key finding with multiple patterns

### Documentation Improvements

- Added i18n usage examples for all supported languages
- Enhanced command help text with translations
- Improved accessibility documentation
- Added real-world monitoring scenarios

## [0.2.0] - 2025-12-20

### Added

- **Phase 1 - Essential Instance Management**: Core infrastructure
- **Feature Roadmap**: Comprehensive development plan

## [0.1.2] - 2025-12-20

### Fixed

- Goreleaser v2 deprecation warnings
- Truffle dependency fully qualified names

## [0.1.1] - 2025-12-19

### Added

- Truffle dependency integration with spawn package

## [Unreleased]

### Planned for 0.4.0

- [ ] Web UI for spawn wizard
- [ ] Auto quota increase requests
- [ ] Cost tracking and budgets
- [ ] Multi-instance orchestration
- [ ] CloudFormation/Terraform output
- [ ] Homebrew formula
- [ ] Chocolatey package

### Planned for 0.5.0

- [ ] Fargate support
- [ ] Lambda integration
- [ ] Custom AMI builder
- [ ] Team/organization features
- [ ] Cross-account quotas

---

## Release Notes

### v0.3.0 - "Global Reach"

This release brings mycelium to a global audience with internationalization, enhanced testing, and powerful new instance management commands.

**Key Highlights:**
- 🌍 **Multilingual**: 6 languages (English, Spanish, French, German, Japanese, Portuguese)
- 📋 **Instance Management**: `spawn list`, `spawn extend`, `spawn ssh` commands
- ✅ **Completion Signals**: Workload-driven lifecycle management
- 🧪 **Comprehensive Testing**: 667+ test cases, 76.8% i18n coverage
- 📚 **Enhanced Documentation**: TESTING.md, MONITORING.md, expanded README
- 🌐 **DNS Gateway**: Lambda-based serverless DNS for spore.host
- 🐛 **Bug Fixes**: Format string safety, i18n improvements

**New Commands:**
```bash
spawn list --family m7i --state running    # Filter instances
spawn extend my-instance 2h                # Extend TTL
spawn ssh i-xxx                            # SSH alias for connect
spored complete --status success           # Signal job completion
```

**Accessibility:**
```bash
spawn --lang es list                       # Spanish
spawn --lang ja connect i-xxx              # Japanese
spawn --accessibility launch               # Screen reader mode
```

**Time to first instance (now in 6 languages!):** 2 minutes for absolute beginners!

**Philosophy:**
Like mycelium spreading across continents, v0.3.0 brings cloud computing to users worldwide in their native language.

### v0.2.0 - "Essential Management"

Phase 1 completion with essential instance management capabilities.

**Key Highlights:**
- Core infrastructure complete
- Roadmap established
- Foundation for advanced features

### v0.1.0 - "The Underground Network"

This is the initial release of mycelium, bringing together truffle and spawn to make AWS EC2 accessible to everyone.

**Key Highlights:**
- 🔍 **truffle** helps you find the right instance (no AWS account needed!)
- 🚀 **spawn** helps you launch it effortlessly (wizard or pipe)
- 🤖 **spored** monitors it automatically (no surprises)
- 🪟 **Windows native** support (finally!)
- 📊 **Quota-aware** (prevents launch failures)
- 💰 **Cost-conscious** (auto-termination, estimates)

**Time to first instance:** 2 minutes for absolute beginners!

**Philosophy:**
Like mycelium in nature connects trees and enables resource sharing, our mycelium connects you to AWS compute resources efficiently and accessibly.

---

[0.3.0]: https://github.com/scttfrdmn/mycelium/releases/tag/v0.3.0
[0.2.0]: https://github.com/scttfrdmn/mycelium/releases/tag/v0.2.0
[0.1.2]: https://github.com/scttfrdmn/mycelium/releases/tag/v0.1.2
[0.1.1]: https://github.com/scttfrdmn/mycelium/releases/tag/v0.1.1
[0.1.0]: https://github.com/scttfrdmn/mycelium/releases/tag/v0.1.0
