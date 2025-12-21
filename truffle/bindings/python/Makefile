# Makefile for building Truffle Python bindings with cgo

.PHONY: all build clean install test

# Detect OS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
    LIB_EXT := dylib
    BUILD_FLAGS := -buildmode=c-shared
else ifeq ($(UNAME_S),Linux)
    LIB_EXT := so
    BUILD_FLAGS := -buildmode=c-shared
else
    LIB_EXT := dll
    BUILD_FLAGS := -buildmode=c-shared
endif

LIB_NAME := libtruffle.$(LIB_EXT)
GO_SRC := truffle/native.go

all: build

build:
	@echo "Building Truffle native library for $(UNAME_S)..."
	cd truffle && go build $(BUILD_FLAGS) -o $(LIB_NAME) native.go
	@echo "Built: truffle/$(LIB_NAME)"

clean:
	rm -f truffle/$(LIB_NAME)
	rm -f truffle/*.h
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "*.pyc" -delete

install: build
	pip install -e .

test: build
	python -m pytest tests/ -v

# Development
dev: build
	pip install -e ".[dev]"

# Build for all platforms (requires cross-compilation setup)
build-all:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -buildmode=c-shared -o truffle/libtruffle_linux_amd64.so $(GO_SRC)
	@echo "Building for macOS (Intel)..."
	GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o truffle/libtruffle_darwin_amd64.dylib $(GO_SRC)
	@echo "Building for macOS (ARM)..."
	GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -o truffle/libtruffle_darwin_arm64.dylib $(GO_SRC)
	@echo "All platforms built!"

.DEFAULT_GOAL := build
