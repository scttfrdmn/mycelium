# Makefile for mycelium suite
.PHONY: all build build-all clean install test help

# Build for current platform
build:
	@echo "Building mycelium suite..."
	@cd truffle && $(MAKE) build
	@cd spawn && $(MAKE) build
	@echo "✅ Build complete!"
	@echo ""
	@echo "Binaries:"
	@echo "  truffle/bin/truffle"
	@echo "  spawn/bin/spawn"
	@echo "  spawn/bin/spawnd"

# Build for all platforms
build-all:
	@echo "Building mycelium suite for all platforms..."
	@cd truffle && $(MAKE) build-all
	@cd spawn && $(MAKE) build-all
	@echo "✅ Build complete for all platforms!"
	@echo ""
	@echo "See truffle/bin/ and spawn/bin/ for binaries"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@cd truffle && $(MAKE) clean
	@cd spawn && $(MAKE) clean
	@echo "✅ Clean complete"

# Install locally (requires sudo)
install:
	@echo "Installing mycelium suite..."
	@cd truffle && $(MAKE) install
	@cd spawn && $(MAKE) install
	@echo "✅ Installed to /usr/local/bin"

# Run tests
test:
	@echo "Running tests..."
	@cd truffle && go test ./...
	@cd spawn && go test ./...
	@echo "✅ Tests passed"

# Show help
help:
	@echo "mycelium - The underground network for AWS compute"
	@echo ""
	@echo "Targets:"
	@echo "  make build       - Build for current platform"
	@echo "  make build-all   - Build for all platforms (Linux, macOS, Windows)"
	@echo "  make clean       - Clean build artifacts"
	@echo "  make install     - Install to /usr/local/bin (requires sudo)"
	@echo "  make test        - Run all tests"
	@echo "  make help        - Show this help"
	@echo ""
	@echo "Quick start:"
	@echo "  1. make build"
	@echo "  2. ./spawn/bin/spawn"
	@echo ""
	@echo "Documentation:"
	@echo "  README.md                - Getting started"
	@echo "  QUICK_REFERENCE.md       - Command cheat sheet"
	@echo "  COMPLETE_ECOSYSTEM.md    - Full overview"

.DEFAULT_GOAL := help
