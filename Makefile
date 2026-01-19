# AI Work Studio Build System
# Simplicity through experience, not complexity through programming

# Project variables
PROJECT_NAME := ai-work-studio
MODULE := github.com/yourusername/ai-work-studio
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.1.0-dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build flags
LDFLAGS := -ldflags="-X $(MODULE)/internal/version.Version=$(VERSION) \
                      -X $(MODULE)/internal/version.BuildTime=$(BUILD_TIME) \
                      -X $(MODULE)/internal/version.Commit=$(COMMIT) \
                      -s -w"

# Go build flags
GOFLAGS := -trimpath

# Binary output directories
BIN_DIR := ./bin
DIST_DIR := ./dist

# Platform-specific binaries
PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64

.PHONY: help build test clean install dev deps check lint format release all

# Default target
all: clean deps test build

help:
	@echo "AI Work Studio Build System"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       Build studio and agent binaries"
	@echo "  test        Run all tests"
	@echo "  clean       Clean build artifacts"
	@echo "  install     Install binaries to \$$GOPATH/bin"
	@echo "  dev         Build for development"
	@echo "  deps        Download dependencies"
	@echo "  check       Run various code quality checks"
	@echo "  lint        Run linting tools"
	@echo "  format      Format code"
	@echo "  release     Build release binaries for all platforms"
	@echo "  help        Show this help message"

# Development build (faster, includes debug info)
dev:
	@echo "Building for development..."
	go build $(GOFLAGS) -o $(BIN_DIR)/ai-work-studio ./cmd/studio
	go build $(GOFLAGS) -o $(BIN_DIR)/ai-work-studio-agent ./cmd/agent
	@echo "Development binaries built in $(BIN_DIR)/"

# Production build
build: deps
	@echo "Building AI Work Studio $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/ai-work-studio ./cmd/studio
	go build $(GOFLAGS) $(LDFLAGS) -o $(BIN_DIR)/ai-work-studio-agent ./cmd/agent
	@echo "Binaries built in $(BIN_DIR)/"
	@echo "  Studio:    $(BIN_DIR)/ai-work-studio"
	@echo "  Agent:     $(BIN_DIR)/ai-work-studio-agent"

# Test all packages
test:
	@echo "Running tests..."
	go test -v ./...
	@echo ""
	@echo "Running integration tests..."
	go test -v ./test/...

# Test with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Code quality checks
check: lint test
	@echo "Running go vet..."
	go vet ./...
	@echo "Checking module consistency..."
	go mod verify

# Lint code
lint:
	@echo "Running linting..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		@echo "golangci-lint not installed. Install with:"; \
		@echo "  go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		go vet ./...; \
	fi

# Format code
format:
	@echo "Formatting code..."
	go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		@echo "goimports not installed. Install with:"; \
		@echo "  go install golang.org/x/tools/cmd/goimports@latest"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BIN_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html
	go clean -cache

# Install to GOPATH/bin
install: build
	@echo "Installing to \$$GOPATH/bin..."
	go install $(LDFLAGS) ./cmd/studio
	go install $(LDFLAGS) ./cmd/agent
	@echo "Installed ai-work-studio and ai-work-studio-agent"

# Release builds for all platforms
release: clean deps test
	@echo "Building release binaries for all platforms..."
	@mkdir -p $(DIST_DIR)
	$(foreach platform, $(PLATFORMS), \
		$(call build_platform,$(platform)))
	@echo "Release binaries built in $(DIST_DIR)/"

# Build function for specific platform
define build_platform
	$(eval GOOS=$(word 1,$(subst -, ,$(1))))
	$(eval GOARCH=$(word 2,$(subst -, ,$(1))))
	$(eval EXT=$(if $(filter windows,$(GOOS)),.exe,))
	@echo "Building $(1)..."
	@mkdir -p $(DIST_DIR)/$(1)
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) $(LDFLAGS) \
		-o $(DIST_DIR)/$(1)/ai-work-studio$(EXT) ./cmd/studio
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build $(GOFLAGS) $(LDFLAGS) \
		-o $(DIST_DIR)/$(1)/ai-work-studio-agent$(EXT) ./cmd/agent
endef

# Quick development cycle
watch:
	@echo "Starting development watch mode..."
	@echo "Install 'entr' for file watching: apt-get install entr (Linux) or brew install entr (macOS)"
	@if command -v entr >/dev/null 2>&1; then \
		find . -name '*.go' | entr -r make dev; \
	else \
		@echo "File watching not available. Use 'make dev' to build manually."; \
	fi

# Create data directory structure
init-data:
	@echo "Initializing data directory structure..."
	@mkdir -p data/nodes data/edges data/backups data/logs data/cache
	@echo "Data directories created in ./data/"

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Commit: $(COMMIT)"
	@echo "Module: $(MODULE)"

# Check if binaries are working
verify: build
	@echo "Verifying builds..."
	./$(BIN_DIR)/ai-work-studio --version 2>/dev/null || echo "Studio binary not ready"
	./$(BIN_DIR)/ai-work-studio-agent --version 2>/dev/null || echo "Agent binary not ready"