# Claude Code Quick Reference - Truffle

## 🚀 Quick Setup (30 seconds)

```bash
cd truffle

# Login to AWS (easiest!)
aws login

# Install dependencies
go mod download

# Build and test
go build -o truffle
./truffle search m7i.large
```

## 📁 Project Layout

```
truffle/
├── cmd/              ← CLI commands (add new commands here)
├── pkg/aws/          ← AWS API logic (EC2 integration)
├── pkg/output/       ← Formatters (add new output formats)
└── examples/         ← Usage scripts (add new examples)
```

## 🎯 Common Claude Code Tasks

### Add a New Filter
```
"Add a --network-performance flag to filter instances by network capability"
→ Modify: pkg/aws/client.go (FilterOptions, matchesFilters)
→ Add flag: cmd/search.go
```

### Add Output Format
```
"Add XML output format support"
→ Modify: pkg/output/printer.go (add PrintXML method)
→ Update: cmd/search.go (switch statement)
```

### Implement Caching
```
"Cache region queries for 5 minutes to speed up repeated searches"
→ Modify: pkg/aws/client.go (add cache map, TTL logic)
→ Add flag: cmd/root.go (--no-cache)
```

### Add Progress Indicator
```
"Show a progress bar when querying multiple regions"
→ Add dependency: go get github.com/schollz/progressbar/v3
→ Modify: pkg/aws/client.go (SearchInstanceTypes)
```

## 💡 Latest Instance Types (Dec 2025)

### Use in Examples
```bash
# General Purpose
m7i.large        # Intel Sapphire Rapids
m8g.large        # Graviton4 (latest!)
m7a.large        # AMD Genoa

# Compute
c7i.xlarge       # Intel compute
c8g.xlarge       # Graviton4 compute
c7a.xlarge       # AMD compute

# Memory
r7i.xlarge       # Intel memory
r8g.xlarge       # Graviton4 memory

# ML
inf2.xlarge      # AWS Inferentia2
trn1.2xlarge     # AWS Trainium
p5.48xlarge      # NVIDIA H100

# Storage
i4i.xlarge       # Local NVMe
```

## 🧪 Testing

```bash
# Quick test
go test ./pkg/aws/

# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose
go test -v ./pkg/aws/
```

## 🔨 Building

```bash
# Simple build
go build -o truffle

# All platforms
make build-all

# With version info
VERSION=0.2.0 make build
```

## 🐛 Debugging

```bash
# Verbose mode
./truffle search m7i.large --verbose

# AWS API debug
export AWS_LOG_LEVEL=debug
./truffle search m7i.large

# Go debugger
dlv debug -- search m7i.large
```

## 📝 Good Claude Code Prompts

### Feature Additions
```
✅ "Add a --compare flag that shows side-by-side comparison of 2 instance types"
✅ "Implement rate limiting for AWS API calls with exponential backoff"
✅ "Add shell completion support for bash and zsh"
✅ "Create a config file at ~/.truffle.yaml for storing default settings"
```

### Improvements
```
✅ "Optimize the concurrent region queries to use worker pools"
✅ "Add retry logic with exponential backoff for failed API calls"
✅ "Implement streaming JSON output for large result sets"
✅ "Add color-coded performance tiers (budget/balanced/performance) in table output"
```

### Bug Fixes
```
✅ "Fix the wildcard matching to handle edge cases like 'm*.large'"
✅ "Handle API throttling errors gracefully with retry"
✅ "Ensure all goroutines are properly cleaned up on context cancellation"
```

### Testing
```
✅ "Add unit tests for the pattern matching logic"
✅ "Create integration tests that mock AWS API responses"
✅ "Add table test cases for the filter matching function"
```

## 📚 Key Files to Know

### cmd/search.go
- Main search logic
- Flag definitions
- Pattern matching
- Output formatting calls

### pkg/aws/client.go
- AWS SDK integration
- EC2 API calls
- Concurrent region queries
- Filtering logic

### pkg/output/printer.go
- All output formatters
- Table, JSON, YAML, CSV
- Color support
- Result grouping

## 🔧 Environment Variables

**Recommended: Use `aws login` (easiest!)**
```bash
aws login
```

**Alternative (manual configuration):**
```bash
# AWS
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxx
AWS_DEFAULT_REGION=us-east-1

# Or use a profile
AWS_PROFILE=my-profile

# Development
TRUFFLE_DEBUG=true
TRUFFLE_VERBOSE=true
```

## 🎨 Code Style

```go
// Good function comment
// GetInstanceTypes retrieves all instance types from the specified region.
// Returns a slice of instance type names or an error.
func (c *Client) GetInstanceTypes(ctx context.Context, region string) ([]string, error) {
    // Implementation
}

// Use meaningful names
instanceType := "m7i.large"  // ✅
it := "m7i.large"            // ❌

// Handle errors properly
if err != nil {
    return fmt.Errorf("failed to describe instances: %w", err)
}
```

## 🚦 Development Workflow

```bash
# 1. Make changes
# 2. Format
go fmt ./...

# 3. Test
go test ./...

# 4. Build
go build -o truffle

# 5. Manual test
./truffle search m7i.large --verbose

# 6. Commit
git add .
git commit -m "feat: add caching support"
```

## 📦 Dependencies

```bash
# Add new dependency
go get github.com/package/name@version

# Update all
go get -u ./...

# Tidy
go mod tidy
```

## 🎁 Quick Wins for Claude Code

Easy enhancements to request:

1. **Add --profile flag** for AWS SSO profiles
2. **Color-code by generation** (7th gen blue, 8th gen green)
3. **Add --sort flag** (by price, vcpu, memory)
4. **Implement --watch mode** to monitor availability
5. **Add instance family icons** in table output (📊 for M, ⚡ for C, 💾 for R)
6. **Create --diff command** to compare two instance types
7. **Add --export-terraform** to generate TF code
8. **Implement result pagination** for large outputs
9. **Add --availability-score** showing AZ coverage
10. **Create --recommend** for instance suggestions

## 📖 Documentation to Read

1. **CLAUDE_CODE_SETUP.md** - Full development guide
2. **INSTANCE_TYPES_2025.md** - Latest instance types
3. **pkg/aws/client.go** - Core AWS logic
4. **cmd/search.go** - Main command

---

**Ready to code!** 🍄 Start Claude Code and begin enhancing Truffle!
