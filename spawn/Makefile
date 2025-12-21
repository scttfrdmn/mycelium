.PHONY: all build build-spawn build-spawnd build-all clean install test

# Version
VERSION ?= 0.1.0

# Build directory
BUILD_DIR = bin

all: build

# Build for current platform
build: build-spawn build-spawnd

build-spawn:
	@echo "Building spawn..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/spawn main.go

build-spawnd:
	@echo "Building spawnd..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/spawnd cmd/spawnd/main.go

# Build for all platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

# Linux AMD64 (x86_64)
build-linux-amd64:
	@echo "Building for Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/spawn-linux-amd64 main.go
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/spawnd-linux-amd64 cmd/spawnd/main.go

# Linux ARM64 (Graviton)
build-linux-arm64:
	@echo "Building for Linux ARM64 (Graviton)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/spawn-linux-arm64 main.go
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/spawnd-linux-arm64 cmd/spawnd/main.go

# macOS AMD64 (Intel)
build-darwin-amd64:
	@echo "Building for macOS AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/spawn-darwin-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/spawnd-darwin-amd64 cmd/spawnd/main.go

# macOS ARM64 (M1/M2)
build-darwin-arm64:
	@echo "Building for macOS ARM64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/spawn-darwin-arm64 main.go
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/spawnd-darwin-arm64 cmd/spawnd/main.go

# Windows AMD64
build-windows-amd64:
	@echo "Building for Windows AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/spawn-windows-amd64.exe main.go
	# Note: spawnd is Linux-only (runs on instances)

# Install locally
install: build
	@echo "Installing spawn and spawnd..."
	@cp $(BUILD_DIR)/spawn /usr/local/bin/
	@cp $(BUILD_DIR)/spawnd /usr/local/bin/
	@echo "Installed to /usr/local/bin/"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Development
dev: build
	@echo "Built for development"

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@mkdir -p release
	tar czf release/spawn-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR) spawn-linux-amd64 spawnd-linux-amd64
	tar czf release/spawn-$(VERSION)-linux-arm64.tar.gz -C $(BUILD_DIR) spawn-linux-arm64 spawnd-linux-arm64
	tar czf release/spawn-$(VERSION)-darwin-amd64.tar.gz -C $(BUILD_DIR) spawn-darwin-amd64 spawnd-darwin-amd64
	tar czf release/spawn-$(VERSION)-darwin-arm64.tar.gz -C $(BUILD_DIR) spawn-darwin-arm64 spawnd-darwin-arm64
	@echo "Release archives created in release/"

.DEFAULT_GOAL := build
