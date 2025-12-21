# 🤖 Claude Code Development Setup

This guide will help you set up Truffle for development using Claude Code.

## Prerequisites

- Go 1.22 or higher installed
- AWS credentials configured
- Git (optional)

## Quick Setup for Claude Code

### 1. Navigate to Project
```bash
cd /path/to/truffle
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Verify Setup
```bash
# Check Go installation
go version

# Login to AWS (easiest method!)
aws login

# Verify AWS credentials
aws sts get-caller-identity
```

**For SSO/IAM Identity Center:**
```bash
aws login --profile my-sso-profile
export AWS_PROFILE=my-sso-profile
```

**Alternative (if not using aws login):**
```bash
# Set environment variables
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
```

### 4. Build and Test
```bash
# Build
go build -o truffle

# Run tests
go test ./...

# Test the CLI
./truffle search m7i.large --regions us-east-1
```

## Project Structure for Development

```
truffle/
├── main.go                    # Entry point - modify to add global features
├── cmd/                       # CLI commands
│   ├── root.go               # Root command - add global flags here
│   ├── search.go             # Search logic - main functionality
│   ├── list.go               # List command - instance type listings
│   └── version.go            # Version info
├── pkg/
│   ├── aws/
│   │   └── client.go         # AWS EC2 API wrapper - core AWS logic
│   └── output/
│       └── printer.go        # Output formatters - add new formats here
└── examples/                  # Usage examples - add new scripts here
```

## Common Development Tasks

### Add a New Command
1. Create new file in `cmd/` directory
2. Define cobra command structure
3. Add to `init()` function to register with root
4. Update README.md with new command docs

### Add a New Output Format
1. Edit `pkg/output/printer.go`
2. Add new method (e.g., `PrintXML()`)
3. Update switch statement in `cmd/search.go`
4. Add flag to `--output` choices

### Add a New Filter
1. Edit `pkg/aws/client.go`
2. Add field to `FilterOptions` struct
3. Implement filter logic in `matchesFilters()`
4. Add flag in `cmd/search.go`

### Add AWS API Features
1. Edit `pkg/aws/client.go`
2. Add new methods to `Client` struct
3. Use AWS SDK v2 patterns
4. Handle pagination if needed

## Testing Your Changes

### Run Unit Tests
```bash
# All tests
go test ./...

# Specific package
go test ./pkg/aws/

# With coverage
go test -cover ./...

# Verbose
go test -v ./...
```

### Manual Testing
```bash
# Rebuild
go build -o truffle

# Test search
./truffle search "m7i.*" --verbose

# Test with filters
./truffle search "m8g.*" --min-vcpu 4 --architecture arm64

# Test output formats
./truffle search m7i.large --output json
./truffle search m7i.large --output yaml
./truffle search m7i.large --output csv
```

### Test with Make
```bash
make test          # Run tests
make build         # Build binary
make lint          # Run linter
make clean         # Clean build artifacts
```

## Development Workflow with Claude Code

### Typical Session
```bash
# Start Claude Code in the project directory
claude-code

# Ask Claude Code to:
# - "Add support for filtering by network performance"
# - "Implement caching for region queries"
# - "Add a compare command to diff instance types"
# - "Fix bug in wildcard matching"
# - "Add progress bar for multi-region queries"
```

### Best Practices for Claude Code Prompts

**Good Prompts:**
- "Add a new flag --network-performance to filter instances by network capability"
- "Implement result caching in pkg/aws/client.go to store region queries for 5 minutes"
- "Add unit tests for the wildcard pattern matching in search.go"
- "Update the table output to include pricing information if available"

**Be Specific:**
- Mention which files to modify
- Reference existing patterns in the codebase
- Ask for tests alongside features
- Request documentation updates

## Instance Types to Use in Examples (December 2025)

### General Purpose
- **m7i.*** - 7th gen Intel (Sapphire Rapids)
- **m7a.*** - 7th gen AMD (Genoa)
- **m8g.*** - 8th gen Graviton4 (ARM)

### Compute Optimized
- **c7i.*** - 7th gen Intel
- **c7a.*** - 7th gen AMD
- **c8g.*** - 8th gen Graviton4
- **c7gn.*** - Network-optimized Graviton3

### Memory Optimized
- **r7i.*** - 7th gen Intel
- **r7a.*** - 7th gen AMD
- **r8g.*** - 8th gen Graviton4

### ML/AI Instances
- **inf2.*** - AWS Inferentia2 (inference)
- **trn1.*** - AWS Trainium (training)
- **trn1n.*** - Trainium with network optimization
- **p5.*** - NVIDIA H100 (latest GPU)
- **g6.*** - NVIDIA L4/L40S (graphics/AI)

### Storage Optimized
- **i4i.*** - 4th gen Intel with NVMe
- **im4gn.*** - Graviton with NVMe
- **is4gen.*** - Graviton SSD

### HPC
- **hpc7g.*** - Graviton3 HPC
- **hpc7a.*** - AMD EPYC HPC

## Environment Variables for Testing

**Recommended: Use `aws login` instead of manual environment variables**
```bash
# Modern approach - no manual config needed!
aws login
```

**If you prefer environment variables:**
```bash
# AWS Configuration
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret

# Or use a profile
export AWS_PROFILE=my-profile

# Go Configuration
export GOARCH=amd64
export GOOS=linux

# Testing
export TRUFFLE_DEBUG=true  # (if you add debug mode)
export TRUFFLE_CACHE_TTL=300  # (if you add caching)
```

## Debugging Tips

### Enable Verbose Output
```bash
./truffle search m7i.large --verbose
```

### Use Go Debugger (Delve)
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug -- search m7i.large

# In delve:
# (dlv) break main.main
# (dlv) continue
# (dlv) print variableName
```

### Add Print Debugging
```go
// In any .go file
fmt.Fprintf(os.Stderr, "DEBUG: variable value: %+v\n", variable)
```

### Check AWS API Calls
```bash
# Enable AWS SDK debug logging
export AWS_LOG_LEVEL=debug
./truffle search m7i.large
```

## Common Issues and Solutions

### "Module not found" errors
```bash
go mod download
go mod tidy
```

### AWS credential errors
```bash
# Check credentials
aws sts get-caller-identity

# Set explicitly
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
```

### Build errors
```bash
# Clean and rebuild
go clean
go build
```

### Test failures
```bash
# Run with verbose output
go test -v ./...

# Run specific test
go test -v -run TestFunctionName ./pkg/aws/
```

## Suggested Enhancements for Claude Code

Here are some features you might want to add with Claude Code:

1. **Result Caching**
   - Cache region queries for 5 minutes
   - Add `--no-cache` flag to bypass

2. **Progress Indicators**
   - Show progress bar for multi-region queries
   - Display "Checking region X of Y..."

3. **Instance Comparison**
   - New `compare` command
   - Side-by-side comparison of instance specs

4. **Pricing Integration**
   - Fetch pricing data from AWS Pricing API
   - Add `--show-pricing` flag

5. **Export to IaC Formats**
   - Generate Terraform resource blocks
   - Output CloudFormation templates

6. **Interactive Mode**
   - TUI with keyboard navigation
   - Filter results interactively

7. **Historical Tracking**
   - Track instance availability over time
   - Notify when new instances are available

8. **Recommendation Engine**
   - Suggest alternative instances
   - Optimize for cost or performance

## Getting Help

- **Documentation**: See README.md, OVERVIEW.md
- **Code Examples**: Check `examples/` directory
- **AWS SDK Docs**: https://aws.github.io/aws-sdk-go-v2/
- **Cobra Docs**: https://cobra.dev/

## Quick Reference

### Build Commands
```bash
go build -o truffle              # Build
make build                       # Build with make
make build-all                   # Multi-platform build
```

### Test Commands
```bash
go test ./...                    # Run all tests
go test -cover ./...             # With coverage
go test -v ./pkg/aws/           # Specific package
```

### Common Claude Code Requests
```
"Add pagination support for large result sets"
"Implement a --quiet mode that only outputs errors"
"Add shell completion for bash and zsh"
"Create a config file at ~/.truffle.yaml for default settings"
"Add support for AWS profiles (--profile flag)"
"Implement parallel AZ queries within each region"
```

---

Ready to develop! Start Claude Code in this directory and begin enhancing Truffle! 🚀
