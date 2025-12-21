# Contributing to Truffle

Thank you for your interest in contributing to Truffle! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and professional. We're all here to make Truffle better!

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check the [existing issues](https://github.com/yourusername/truffle/issues)
2. Ensure you're using the latest version
3. Collect relevant information (OS, Go version, AWS region, etc.)

Create a bug report including:
- Clear, descriptive title
- Steps to reproduce
- Expected vs actual behavior
- Error messages and logs
- Environment details

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:
1. Use a clear, descriptive title
2. Provide detailed description of the proposed feature
3. Explain why this would be useful
4. Include examples if possible

### Pull Requests

1. **Fork and Clone**
   ```bash
   git clone https://github.com/yourusername/truffle.git
   cd truffle
   ```

2. **Create a Branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```

3. **Make Your Changes**
   - Follow the code style (run `go fmt`)
   - Add tests for new features
   - Update documentation as needed

4. **Test Your Changes**
   ```bash
   # Run tests
   make test
   
   # Run linter
   make lint
   
   # Build and test locally
   make build
   ./bin/truffle search m5.large --verbose
   ```

5. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: add amazing feature"
   ```
   
   Use conventional commit format:
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation changes
   - `test:` - Test changes
   - `refactor:` - Code refactoring
   - `chore:` - Maintenance tasks

6. **Push and Create PR**
   ```bash
   git push origin feature/amazing-feature
   ```
   
   Then create a Pull Request on GitHub.

## Development Setup

### Prerequisites

- Go 1.22 or higher
- Make (optional)
- golangci-lint (for linting)
- AWS account (for testing)

### Initial Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/truffle.git
cd truffle

# Install dependencies
go mod download

# Build
make build

# Run tests
make test
```

### Project Structure

```
truffle/
├── main.go              # Entry point
├── cmd/                 # CLI commands
│   ├── root.go         # Root command
│   ├── search.go       # Search command
│   ├── list.go         # List command
│   └── version.go      # Version command
├── pkg/
│   ├── aws/            # AWS client
│   │   └── client.go
│   └── output/         # Output formatters
│       └── printer.go
├── examples/           # Example scripts
├── Makefile           # Build automation
└── README.md
```

## Code Style

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `go fmt` for formatting
- Run `golangci-lint` before committing
- Keep functions focused and small
- Use meaningful variable names
- Add comments for exported functions

### Example

```go
// GetInstanceTypes retrieves all instance types available in the specified region.
// It returns a slice of instance type names or an error if the operation fails.
func (c *Client) GetInstanceTypes(ctx context.Context, region string) ([]string, error) {
    // Implementation
}
```

## Testing

### Writing Tests

Add tests for new features:

```go
// pkg/aws/client_test.go
func TestExtractFamily(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"m5.large", "m5"},
        {"c5.xlarge", "c5"},
        {"invalid", "invalid"},
    }
    
    for _, tt := range tests {
        t.Run(tt.input, func(t *testing.T) {
            result := extractFamily(tt.input)
            if result != tt.expected {
                t.Errorf("got %s, want %s", result, tt.expected)
            }
        })
    }
}
```

### Running Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./pkg/aws/

# Verbose
go test -v ./...
```

## Documentation

### Code Documentation

- Add comments to all exported functions
- Use GoDoc format
- Include examples when helpful

### User Documentation

When adding features, update:
- README.md (usage examples)
- INSTALL.md (if installation changes)
- Examples in examples/ directory

## Release Process

Maintainers handle releases:

1. Update version in `cmd/version.go`
2. Update CHANGELOG.md
3. Create git tag: `git tag v0.2.0`
4. Push tag: `git push origin v0.2.0`
5. GitHub Actions builds and publishes release

## Getting Help

- 💬 [GitHub Discussions](https://github.com/yourusername/truffle/discussions)
- 🐛 [Issue Tracker](https://github.com/yourusername/truffle/issues)
- 📧 Email: maintainer@example.com

## Recognition

Contributors will be:
- Listed in CONTRIBUTORS.md
- Mentioned in release notes
- Credited in the project

Thank you for making Truffle better! 🍄
