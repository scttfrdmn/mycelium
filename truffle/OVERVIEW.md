# Truffle - Project Overview

## 📋 Executive Summary

Truffle is a production-ready, Go-based CLI tool designed to help AWS users discover which EC2 instance types are available across different AWS regions and availability zones. It solves the common problem of "where can I deploy this instance type?" with an elegant, fast, and developer-friendly interface.

## 🎯 Problem Statement

When planning AWS deployments, engineers often need to:
- Verify instance type availability across multiple regions
- Plan multi-region deployments with consistent instance types
- Optimize costs by finding alternative instance types
- Validate infrastructure-as-code configurations
- Ensure capacity planning across availability zones

Manually checking this information through the AWS Console or CLI is tedious and error-prone. Truffle automates this process.

## ✨ Key Features

### Core Functionality
- **Wildcard Search**: Pattern matching with `*` and `?` for flexible queries
- **Multi-Region**: Query all AWS regions or filter specific ones concurrently
- **AZ Details**: Optional availability zone information
- **Rich Filtering**: Filter by architecture, vCPUs, memory, instance family

### Output Formats
- **Table**: Colorful, human-readable terminal output
- **JSON**: Structured data for automation and scripting
- **YAML**: Clean, readable format for configuration
- **CSV**: Spreadsheet-compatible for analysis

### Performance
- **Concurrent Queries**: Parallel region scanning for speed
- **Configurable Timeout**: Adjustable API timeout settings
- **Efficient Caching**: (Planned feature)

### Developer Experience
- **Intuitive CLI**: Cobra-powered command structure
- **Comprehensive Help**: Built-in documentation
- **Verbose Mode**: Detailed logging for debugging
- **Exit Codes**: Proper status codes for automation

## 🏗️ Architecture

### Technology Stack
- **Language**: Go 1.22+
- **CLI Framework**: Cobra
- **AWS SDK**: AWS SDK for Go v2
- **Output Formatting**: 
  - tablewriter (tables)
  - encoding/json (JSON)
  - gopkg.in/yaml.v3 (YAML)
  - encoding/csv (CSV)
- **Colors**: fatih/color

### Project Structure
```
truffle/
├── main.go                    # Application entry point
├── cmd/                       # CLI commands
│   ├── root.go               # Root command & global flags
│   ├── search.go             # Search command
│   ├── list.go               # List command
│   └── version.go            # Version command
├── pkg/                       # Core packages
│   ├── aws/                  # AWS client implementation
│   │   └── client.go         # EC2 API wrapper
│   └── output/               # Output formatters
│       └── printer.go        # Multi-format printer
├── examples/                  # Usage examples
│   ├── multi-region-check.sh
│   ├── terraform-integration.sh
│   ├── compare-instances.sh
│   └── cicd-validation.sh
├── .github/workflows/        # CI/CD
│   ├── ci.yml               # Continuous integration
│   └── release.yml          # Release automation
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── Makefile                  # Build automation
├── README.md                 # Main documentation
├── QUICKSTART.md            # Quick start guide
├── INSTALL.md               # Installation instructions
├── CONTRIBUTING.md          # Contribution guidelines
└── LICENSE                  # MIT License
```

### Design Patterns
- **Command Pattern**: Clean separation of CLI commands
- **Concurrent Processing**: Goroutines for parallel region queries
- **Strategy Pattern**: Pluggable output formatters
- **Builder Pattern**: Structured result construction

## 🔧 Technical Implementation

### AWS API Integration
```go
// Core AWS client functionality
- DescribeRegions: Get all AWS regions
- DescribeInstanceTypes: List instance types per region
- DescribeInstanceTypeOfferings: Get AZ availability
```

### Search Algorithm
1. Parse wildcard pattern to regex
2. Get target regions (all or filtered)
3. Query each region concurrently (max 10 concurrent)
4. Match instance types against pattern
5. Apply filters (architecture, vCPUs, memory)
6. Optionally fetch AZ details
7. Sort and format results

### Performance Optimizations
- Concurrent region queries with semaphore (limit: 10)
- Context-based timeout management
- Efficient result aggregation with mutexes
- Streaming output for large datasets

## 📊 Use Cases

### 1. Multi-Region Deployment Planning
**Scenario**: Deploy application across 3 regions with m5.large
```bash
truffle search m5.large --regions us-east-1,eu-west-1,ap-southeast-1 --include-azs
```

### 2. Cost Optimization
**Scenario**: Compare Intel vs AMD instance types
```bash
truffle search "m5.xlarge" --output csv > intel.csv
truffle search "m5a.xlarge" --output csv > amd.csv
```

### 3. Terraform Integration
**Scenario**: Generate list of available regions for Terraform
```bash
truffle search c5.xlarge --output json | jq -r '.[].region' > regions.txt
```

### 4. CI/CD Validation
**Scenario**: Validate instance availability before deployment
```bash
# In CI pipeline
if ! truffle search t3.medium --regions $TARGET_REGION --output json | jq -e 'length > 0'; then
  echo "Instance type not available"
  exit 1
fi
```

### 5. Capacity Planning
**Scenario**: Find all instances with specific resource requirements
```bash
truffle search "*" --min-vcpu 8 --min-memory 32 --architecture arm64
```

## 🚀 Development Roadmap

### Version 0.1.0 (Current)
- ✅ Basic search functionality
- ✅ Multiple output formats
- ✅ Wildcard pattern matching
- ✅ Region filtering
- ✅ AZ details
- ✅ Comprehensive documentation

### Version 0.2.0 (Planned)
- [ ] Results caching for faster queries
- [ ] Instance type comparison feature
- [ ] Price information integration
- [ ] Interactive TUI mode
- [ ] Configuration file support

### Version 0.3.0 (Future)
- [ ] AWS China & GovCloud support
- [ ] Historical availability tracking
- [ ] Export to IaC formats (Terraform, CloudFormation)
- [ ] Instance type recommendation engine
- [ ] Performance benchmarking data

### Version 1.0.0 (Goals)
- [ ] Full test coverage (>80%)
- [ ] Production hardening
- [ ] Performance optimization
- [ ] Comprehensive logging
- [ ] Monitoring integration

## 📈 Success Metrics

### Technical Metrics
- Query time: <30s for all regions
- Concurrent requests: Up to 10 regions simultaneously
- Memory usage: <100MB typical
- Binary size: <20MB
- Test coverage: Target 80%+

### User Metrics
- Installation time: <5 minutes
- Time to first query: <1 minute
- Learning curve: <15 minutes
- Documentation completeness: 100%

## 🔒 Security Considerations

### AWS Credentials
- Supports AWS SDK credential chain
- No credentials stored in binary
- IAM role support for EC2/ECS/Lambda
- Environment variable support
- Credentials file support

### Required Permissions (Read-Only)
```json
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": [
      "ec2:DescribeRegions",
      "ec2:DescribeInstanceTypes",
      "ec2:DescribeInstanceTypeOfferings"
    ],
    "Resource": "*"
  }]
}
```

### Security Best Practices
- Minimal AWS permissions required
- No write operations
- No data stored locally
- No external dependencies beyond AWS SDK
- Supply chain security via Go modules

## 🧪 Testing Strategy

### Unit Tests
- AWS client mocking
- Output formatter testing
- Pattern matching validation
- Filter logic testing

### Integration Tests
- Real AWS API calls (in CI only)
- Multi-region testing
- Error handling validation
- Timeout behavior

### End-to-End Tests
- CLI command testing
- Output format validation
- Real-world scenario simulation

## 📦 Distribution

### Binary Releases
- GitHub Releases with checksums
- Pre-built binaries for:
  - Linux (amd64, arm64)
  - macOS (amd64, arm64)
  - Windows (amd64)

### Package Managers (Future)
- Homebrew (macOS/Linux)
- apt/yum repositories
- Snap package
- Docker image

## 🤝 Contributing

We welcome contributions! Areas of focus:
- Performance improvements
- New output formats
- Additional filtering options
- Bug fixes and documentation
- Example scripts and use cases

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## 📄 License

MIT License - See [LICENSE](LICENSE) file

## 🙏 Acknowledgments

- AWS SDK for Go team
- Cobra CLI framework
- Go community
- Open source contributors

## 📞 Support & Community

- **Documentation**: README.md, QUICKSTART.md, INSTALL.md
- **Issues**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Examples**: examples/ directory
- **Updates**: GitHub Releases

---

**Status**: Active Development
**Stability**: Beta
**Version**: 0.1.0
**Last Updated**: December 2024

Made with ❤️ for the AWS community
