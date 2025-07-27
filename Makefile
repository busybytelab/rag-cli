.PHONY: build build-all clean test unit-test integration-test install deps lint format

# Binary name
BINARY_NAME=rag-cli
BUILD_DIR=build

# Version from git tag, default to dev
VERSION ?= $(shell git describe --tags 2>/dev/null || echo "dev")

# Build for the current platform
build:
	@echo "Building for current platform..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "-X github.com/busybytelab.com/rag-cli/cmd.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for all supported platforms
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/busybytelab.com/rag-cli/cmd.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	
	@echo "Building for Linux (arm64)..."
	@GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/busybytelab.com/rag-cli/cmd.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/busybytelab.com/rag-cli/cmd.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/busybytelab.com/rag-cli/cmd.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/busybytelab.com/rag-cli/cmd.Version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# Clean build directory
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Run unit tests (fast, no external dependencies)
unit-test:
	@echo "Running unit tests..."
	@go test -v -count=1 -race -short -timeout=5m ./...

# Run integration tests (may make external network calls)
integration-test:
	@echo "Running integration tests..."
	@go test -v -count=1 -race -tags=integration -timeout=10m ./...

# Run all tests (both unit and integration)
test: unit-test integration-test
	@echo "All tests completed successfully!"

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, skipping linting"; \
	fi

# Format code
format:
	@echo "Formatting code..."
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	else \
		echo "goimports not found, skipping imports formatting"; \
	fi

# Install the binary
install: build
	@echo "Installing binary to /usr/local/bin/$(BINARY_NAME)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete!"

# Development setup
dev-setup: deps
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@echo "Development setup complete!"

# Run with hot reload (requires air)
dev:
	@if command -v air >/dev/null 2>&1; then \
		air; \
	else \
		echo "Air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "Running without hot reload..."; \
		go run .; \
	fi
