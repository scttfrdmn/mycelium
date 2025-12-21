# Build & Deployment Guide

This guide covers building, testing, and deploying Truffle in various environments.

## 🔨 Building from Source

### Prerequisites
```bash
# Check Go version (need 1.22+)
go version

# Install make (optional but recommended)
# macOS
brew install make

# Linux (Debian/Ubuntu)
sudo apt-get install make
```

### Quick Build
```bash
# Clone repository
git clone https://github.com/yourusername/truffle.git
cd truffle

# Download dependencies
go mod download

# Build binary
go build -o truffle

# Test it
./truffle version
```

### Using Makefile
```bash
# Download dependencies
make deps

# Build
make build

# Install system-wide
sudo make install

# Run tests
make test

# Format code
make fmt

# Run linter
make lint
```

### Build for Multiple Platforms
```bash
# Build all platforms
make build-all

# Manual cross-compilation
GOOS=linux GOARCH=amd64 go build -o truffle-linux-amd64
GOOS=darwin GOARCH=arm64 go build -o truffle-darwin-arm64
GOOS=windows GOARCH=amd64 go build -o truffle-windows-amd64.exe
```

### Build with Version Information
```bash
VERSION="0.1.0"
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS="-X github.com/yourusername/truffle/cmd.Version=${VERSION} \
         -X github.com/yourusername/truffle/cmd.GitCommit=${COMMIT} \
         -X github.com/yourusername/truffle/cmd.BuildDate=${DATE} \
         -s -w"

go build -ldflags "${LDFLAGS}" -o truffle
```

## 🧪 Testing

### Run All Tests
```bash
go test ./...
```

### Run with Coverage
```bash
go test -cover ./...

# Generate HTML coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Run with Race Detector
```bash
go test -race ./...
```

### Test Specific Package
```bash
go test ./pkg/aws/
go test ./pkg/output/
go test ./cmd/
```

### Verbose Testing
```bash
go test -v ./...
```

## 📦 Packaging

### Create Release Binary
```bash
# Clean build
make clean

# Build optimized binary
go build -ldflags="-s -w" -trimpath -o truffle

# Verify binary
./truffle version
ls -lh truffle
```

### Create Distribution Archive
```bash
VERSION="0.1.0"
PLATFORM="linux-amd64"

# Create archive
tar czf truffle-${VERSION}-${PLATFORM}.tar.gz \
    truffle \
    README.md \
    LICENSE \
    QUICKSTART.md

# Create checksum
sha256sum truffle-${VERSION}-${PLATFORM}.tar.gz > \
    truffle-${VERSION}-${PLATFORM}.tar.gz.sha256
```

## 🚀 Deployment Strategies

### 1. Direct Installation
```bash
# Build and install
go build -o truffle
sudo mv truffle /usr/local/bin/

# Verify
truffle version
```

### 2. Docker Container
```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o truffle

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/truffle /usr/local/bin/

ENTRYPOINT ["truffle"]
```

Build and run:
```bash
docker build -t truffle:latest .
docker run --rm \
    -e AWS_ACCESS_KEY_ID=xxx \
    -e AWS_SECRET_ACCESS_KEY=xxx \
    truffle:latest search m5.large
```

### 3. AWS Lambda
```bash
# Build for Lambda
GOOS=linux GOARCH=amd64 go build -o bootstrap

# Create deployment package
zip function.zip bootstrap

# Deploy (using AWS CLI)
aws lambda create-function \
    --function-name truffle-search \
    --runtime provided.al2 \
    --handler bootstrap \
    --zip-file fileb://function.zip \
    --role arn:aws:iam::ACCOUNT:role/lambda-role
```

### 4. CI/CD Pipeline

#### GitHub Actions
```yaml
# .github/workflows/build.yml
name: Build and Test

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Build
        run: go build -v ./...
      
      - name: Test
        run: go test -v ./...
      
      - name: Upload binary
        uses: actions/upload-artifact@v4
        with:
          name: truffle
          path: truffle
```

#### GitLab CI
```yaml
# .gitlab-ci.yml
stages:
  - build
  - test

build:
  stage: build
  image: golang:1.22
  script:
    - go build -o truffle
  artifacts:
    paths:
      - truffle

test:
  stage: test
  image: golang:1.22
  script:
    - go test -v ./...
```

### 5. Package Managers (Future)

#### Homebrew
```ruby
# Formula/truffle.rb
class Truffle < Formula
  desc "AWS EC2 instance type region finder"
  homepage "https://github.com/yourusername/truffle"
  url "https://github.com/yourusername/truffle/archive/v0.1.0.tar.gz"
  sha256 "..."

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"truffle"
  end

  test do
    system "#{bin}/truffle", "version"
  end
end
```

Install:
```bash
brew tap yourusername/tap
brew install truffle
```

## 🔧 Development Setup

### Local Development
```bash
# Clone and setup
git clone https://github.com/yourusername/truffle.git
cd truffle

# Install dependencies
go mod download

# Run without building
go run . search m5.large

# Watch for changes (with entr)
ls **/*.go | entr -r go run . search m5.large
```

### Using Air (Live Reload)
```bash
# Install air
go install github.com/cosmtrek/air@latest

# Create .air.toml
cat > .air.toml << 'EOF'
root = "."
tmp_dir = "tmp"

[build]
  cmd = "go build -o ./tmp/truffle ."
  bin = "tmp/truffle"
  include_ext = ["go"]
  exclude_dir = ["tmp", "vendor"]
EOF

# Run with live reload
air
```

## 📊 Performance Optimization

### Binary Size Reduction
```bash
# Use flags to reduce size
go build -ldflags="-s -w" -trimpath -o truffle

# Further compress with upx
upx --best --lzma truffle
```

### Profiling
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof
```

## 🔒 Security Hardening

### Secure Build Flags
```bash
go build \
    -trimpath \
    -ldflags="-s -w -X main.version=0.1.0 -extldflags '-static'" \
    -o truffle
```

### Vulnerability Scanning
```bash
# Check for known vulnerabilities
go list -json -m all | nancy sleuth

# Using govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

## 📝 Release Checklist

- [ ] Update version in `cmd/version.go`
- [ ] Update `README.md` with new features
- [ ] Update `CHANGELOG.md`
- [ ] Run full test suite: `make test`
- [ ] Run linter: `make lint`
- [ ] Build for all platforms: `make build-all`
- [ ] Create git tag: `git tag v0.1.0`
- [ ] Push tag: `git push origin v0.1.0`
- [ ] Create GitHub release with binaries
- [ ] Update documentation
- [ ] Announce release

## 🐛 Debugging

### Enable Verbose Logging
```bash
truffle search m5.large --verbose
```

### Debug with Delve
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug
dlv debug -- search m5.large
```

### Trace AWS API Calls
```bash
# Set AWS SDK logging
export AWS_SDK_LOAD_CONFIG=1
export AWS_LOG_LEVEL=debug

truffle search m5.large
```

## 💡 Tips & Best Practices

1. **Always run tests before committing**
   ```bash
   make test lint
   ```

2. **Use version tags for dependencies**
   ```bash
   go get github.com/package@v1.2.3
   ```

3. **Keep dependencies updated**
   ```bash
   go get -u ./...
   go mod tidy
   ```

4. **Document all exported functions**
   ```go
   // GetInstanceTypes retrieves instance types from AWS.
   func GetInstanceTypes() {}
   ```

5. **Use meaningful commit messages**
   ```bash
   git commit -m "feat: add ARM64 architecture filter"
   ```

## 📚 Additional Resources

- [Go Release Guide](https://golang.org/doc/devel/release.html)
- [Effective Go](https://golang.org/doc/effective_go.html)
- [AWS SDK for Go](https://aws.github.io/aws-sdk-go-v2/)
- [Cobra Documentation](https://cobra.dev/)

---

For questions or issues, open a ticket on GitHub!
