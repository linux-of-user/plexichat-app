# PlexiChat Client Makefile

# Build configuration
BINARY_NAME=plexichat
GUI_BINARY_NAME=plexichat-gui
BUILD_DIR=build
VERSION=1.0.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# Go build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME) -s -w"
CGO_ENABLED=1

# Default target
.PHONY: all
all: clean build

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Build CLI version
.PHONY: cli
cli: $(BUILD_DIR)
	@echo "Building CLI client..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/plexichat

# Build GUI version
.PHONY: gui
gui: $(BUILD_DIR)
	@echo "Building GUI client..."
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(GUI_BINARY_NAME) plexichat-gui.go

# Build both versions
.PHONY: build
build: cli gui

# Test the CLI
.PHONY: test-cli
test-cli: cli
	@echo "Testing CLI client..."
	$(BUILD_DIR)/$(BINARY_NAME) version
	$(BUILD_DIR)/$(BINARY_NAME) help

# Test client functionality
.PHONY: test-client
test-client:
	@echo "Running client tests..."
	go run tests/client_functionality_test.go

# Run all tests
.PHONY: test
test: test-client test-cli

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run

# Development build (with debug info)
.PHONY: dev
dev: $(BUILD_DIR)
	@echo "Building development version..."
	CGO_ENABLED=$(CGO_ENABLED) go build -gcflags="all=-N -l" -o $(BUILD_DIR)/$(BINARY_NAME)-dev ./cmd/plexichat

# Cross-platform builds
.PHONY: build-windows
build-windows: $(BUILD_DIR)
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows.exe ./cmd/plexichat

.PHONY: build-linux
build-linux: $(BUILD_DIR)
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux ./cmd/plexichat

.PHONY: build-macos
build-macos: $(BUILD_DIR)
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-macos ./cmd/plexichat

.PHONY: build-all
build-all: build-windows build-linux build-macos

# Install locally
.PHONY: install
install: cli
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	go install $(LDFLAGS) ./cmd/plexichat

# Show help
.PHONY: help
help:
	@echo "PlexiChat Client Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  all         - Clean and build both CLI and GUI"
	@echo "  cli         - Build CLI client only"
	@echo "  gui         - Build GUI client only"
	@echo "  build       - Build both CLI and GUI"
	@echo "  test        - Run all tests"
	@echo "  test-cli    - Test CLI functionality"
	@echo "  test-client - Run client unit tests"
	@echo "  clean       - Remove build artifacts"
	@echo "  deps        - Install dependencies"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  dev         - Build development version with debug info"
	@echo "  install     - Install CLI to GOPATH/bin"
	@echo "  build-all   - Build for all platforms"
	@echo "  help        - Show this help"
	@echo ""
	@echo "Cross-platform builds:"
	@echo "  build-windows - Build for Windows"
	@echo "  build-linux   - Build for Linux"
	@echo "  build-macos   - Build for macOS"
	@echo ""
	@echo "Environment variables:"
	@echo "  CGO_ENABLED - Enable/disable CGO (default: 1)"
	@echo "  VERSION     - Build version (default: $(VERSION))"
