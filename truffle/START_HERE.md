# 🍄 Truffle - Ready for Claude Code Development

This is the complete Truffle project, optimized for development with **Claude Code**.

## 🎯 What is Truffle?

A Go-based CLI tool to find AWS EC2 instance types across regions and availability zones.

**Example:**
```bash
$ truffle search m7i.large
🍄 Found 1 instance type(s) across 17 region(s)

+---------------+-----------+-------+--------------+--------------+
| Instance Type | Region    | vCPUs | Memory (GiB) | Architecture |
+---------------+-----------+-------+--------------+--------------+
| m7i.large     | us-east-1 |     2 |          8.0 | x86_64      |
| ...           | ...       | ...   | ...          | ...         |
+---------------+-----------+-------+--------------+--------------+
```

## 🚀 Setup for Claude Code (3 steps)

### 1. Configure AWS Credentials
```bash
cd truffle

# Easiest method - just login!
aws login
```

**That's it for AWS!** ✨ The `aws login` command:
- Handles authentication automatically
- Works with MFA
- Perfect for SSO/IAM Identity Center  
- Refreshes credentials automatically

**For SSO users:**
```bash
aws login --profile my-sso-profile
export AWS_PROFILE=my-sso-profile
```

**Alternative (if not using aws login):**
```bash
cp .env.example .env
# Edit .env with your AWS keys, then:
source .env
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Build & Test
```bash
go build -o truffle
./truffle search m7i.large
```

**Done!** Now you can develop with Claude Code. 🎉

## 📚 Documentation Quick Links

| File | What it is |
|------|------------|
| **CLAUDE_CODE_QUICK_REF.md** | ⭐ Quick reference for Claude Code |
| **CLAUDE_CODE_SETUP.md** | Full development setup guide |
| **INSTANCE_TYPES_2025.md** | Latest AWS instance types (Dec 2025) |
| **README.md** | Complete feature documentation |
| **QUICKSTART.md** | 5-minute user guide |

## 🤖 Using Claude Code

### Start Development
```bash
cd truffle
claude-code
```

### Example Prompts for Claude Code

**Add Features:**
```
"Add caching to store region queries for 5 minutes"
"Implement a progress bar for multi-region searches"
"Add a --compare flag to show side-by-side instance specs"
"Create a TUI mode with interactive filtering"
```

**Improve Code:**
```
"Optimize concurrent region queries with worker pools"
"Add retry logic with exponential backoff for API calls"
"Implement streaming output for large result sets"
"Add comprehensive error handling"
```

**Add Tests:**
```
"Create unit tests for wildcard pattern matching"
"Add integration tests with mocked AWS responses"
"Implement table-driven tests for filter logic"
```

**Fix Issues:**
```
"Fix edge case in wildcard matching for patterns like 'm*.large'"
"Handle API throttling with proper retry mechanism"
"Ensure goroutines are cleaned up on timeout"
```

## 📁 Project Structure

```
truffle/
├── main.go                      # Entry point
├── cmd/                         # CLI commands
│   ├── root.go                 # Root command + global flags
│   ├── search.go               # Search logic (main feature)
│   ├── list.go                 # List instance families
│   └── version.go              # Version info
├── pkg/
│   ├── aws/client.go           # AWS EC2 API integration
│   └── output/printer.go       # Output formatters
├── examples/                    # Usage examples
├── .env.example                # Environment template
└── [documentation files]
```

## 🎯 Latest Instance Types (Dec 2025)

**All examples updated with latest AWS instances:**

| Family | Latest | Use Case |
|--------|--------|----------|
| General | **m8g** (Graviton4) | Best price-performance |
| General | **m7i** (Intel 7th) | Latest Intel |
| Compute | **c8g** (Graviton4) | Compute intensive |
| Memory | **r8g** (Graviton4) | Memory intensive |
| ML Inference | **inf2** | AWS Inferentia2 |
| ML Training | **trn1** | AWS Trainium |
| GPU | **p5** | NVIDIA H100 |

See **INSTANCE_TYPES_2025.md** for complete list.

## 🔧 Development Commands

```bash
# Build
go build -o truffle

# Build all platforms
make build-all

# Test
go test ./...

# Test with coverage
make test-coverage

# Format code
go fmt ./...

# Run linter
make lint
```

## 💡 Quick Wins for Development

Easy enhancements you can ask Claude Code to implement:

1. ✨ **Add AWS SSO profile support** (`--profile` flag)
2. 🎨 **Color-code by instance generation** (7th gen vs 8th gen)
3. 📊 **Add sorting options** (`--sort-by price|vcpu|memory`)
4. 👀 **Watch mode** to monitor instance availability
5. 🔍 **Instance comparison** (`truffle compare m7i.large m8g.large`)
6. 📝 **Export to Terraform** format
7. ⚡ **Result caching** with configurable TTL
8. 📈 **Availability scoring** (how many AZs/regions)
9. 💰 **Price information** from AWS Pricing API
10. 🤖 **Instance recommendations** based on workload

## 🧪 Testing Your Changes

```bash
# After making changes with Claude Code:

# 1. Run tests
go test ./...

# 2. Build
go build -o truffle

# 3. Manual test
./truffle search m7i.large --verbose

# 4. Test different outputs
./truffle search "m8g.*" --output json
./truffle search "c7i.*" --output yaml
./truffle search "r8g.*" --output csv

# 5. Test filters
./truffle search "*" --min-vcpu 8 --min-memory 32
./truffle search "*" --architecture arm64
```

## 🎓 Learning the Codebase

### Start Here
1. **Read cmd/search.go** - Main search command
2. **Read pkg/aws/client.go** - AWS integration
3. **Read pkg/output/printer.go** - Output formatting

### Key Functions
- `SearchInstanceTypes()` - Main search logic
- `matchesFilters()` - Instance filtering
- `PrintTable()` - Table output formatting

### Key Patterns
- **Cobra** for CLI structure
- **Concurrent queries** with goroutines
- **Context** for timeouts
- **AWS SDK v2** for API calls

## 📦 Dependencies

Main dependencies (already in go.mod):
- `github.com/spf13/cobra` - CLI framework
- `github.com/aws/aws-sdk-go-v2` - AWS SDK
- `github.com/fatih/color` - Terminal colors
- `github.com/olekukonko/tablewriter` - Tables
- `gopkg.in/yaml.v3` - YAML output

## 🔒 AWS Permissions Required

Truffle needs these read-only permissions:
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

## 🎯 Project Status

- ✅ **Production Ready** - Fully functional CLI
- ✅ **Well Documented** - Comprehensive docs
- ✅ **Test Coverage** - Unit tests included
- ✅ **CI/CD Ready** - GitHub Actions workflows
- ✅ **Updated for 2025** - Latest instance types

## 🤝 Contributing Ideas

Areas perfect for Claude Code to enhance:

- **Performance**: Caching, connection pooling, parallel optimization
- **Features**: New commands, filters, output formats
- **UX**: Progress bars, interactive mode, better error messages
- **Integration**: Terraform export, pricing API, CloudFormation
- **Testing**: More test coverage, integration tests
- **Documentation**: Code examples, tutorials

## 📞 Getting Help

- 📖 **CLAUDE_CODE_QUICK_REF.md** - Quick reference
- 📘 **CLAUDE_CODE_SETUP.md** - Full setup guide
- 📗 **README.md** - Feature documentation
- 📙 **INSTANCE_TYPES_2025.md** - Instance types reference

## ⚡ Ready to Start!

```bash
# 1. Navigate to project
cd truffle

# 2. Login to AWS (super easy!)
aws login

# 3. Install dependencies
go mod download

# 4. Build
go build -o truffle

# 5. Test
./truffle search m7i.large

# 6. Start developing with Claude Code
claude-code
```

**Ask Claude Code to build something amazing!** 🚀

---

Made with ❤️ for development with Claude Code
Updated: December 2025 with latest AWS instance types
