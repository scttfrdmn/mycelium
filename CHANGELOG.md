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

## [Unreleased]

### Planned for 0.2.0

- [ ] Web UI for spawn wizard
- [ ] Auto quota increase requests
- [ ] Cost tracking and budgets
- [ ] Multi-instance orchestration
- [ ] CloudFormation/Terraform output
- [ ] Homebrew formula
- [ ] Chocolatey package

### Planned for 0.3.0

- [ ] Fargate support
- [ ] Lambda integration
- [ ] Custom AMI builder
- [ ] Team/organization features
- [ ] Cross-account quotas

---

## Release Notes

### v0.1.0 - "The Underground Network"

This is the initial release of mycelium, bringing together truffle and spawn to make AWS EC2 accessible to everyone.

**Key Highlights:**
- 🔍 **truffle** helps you find the right instance (no AWS account needed!)
- 🚀 **spawn** helps you launch it effortlessly (wizard or pipe)
- 🤖 **spawnd** monitors it automatically (no surprises)
- 🪟 **Windows native** support (finally!)
- 📊 **Quota-aware** (prevents launch failures)
- 💰 **Cost-conscious** (auto-termination, estimates)

**Time to first instance:** 2 minutes for absolute beginners!

**Philosophy:**
Like mycelium in nature connects trees and enables resource sharing, our mycelium connects you to AWS compute resources efficiently and accessibly.

---

[0.1.0]: https://github.com/yourname/mycelium/releases/tag/v0.1.0
