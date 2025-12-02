.PHONY: help build test lint install clean fmt vet run

# Variables
BINARY_NAME=kilt
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary for current platform
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/kilt
	@echo "Binary created: bin/$(BINARY_NAME)"

build-all: ## Build binaries for all supported platforms
	@echo "Building for all platforms..."
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/kilt
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/kilt
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/kilt
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/kilt
	@echo "All binaries created in bin/"

test: ## Run tests
	go test -v -race -coverprofile=coverage.out ./...

test-integration: ## Run integration tests
	go test -v -race ./test/integration/...

test-all: test test-integration ## Run all tests (unit + integration)

test-coverage: test ## Run tests with coverage report
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: ## Run linters
	@echo "Running golangci-lint..."
	golangci-lint run ./...

lint-install: ## Install golangci-lint
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.60.1)

fmt: ## Format code with gofmt and goimports
	go fmt ./...
	@if command -v goimports > /dev/null 2>&1; then \
		echo "Running goimports..."; \
		goimports -w -local github.com/unravelling/kilt .; \
	else \
		echo "Warning: goimports not found. Install with: go install golang.org/x/tools/cmd/goimports@latest"; \
	fi
	@echo "Code formatted"

vet: ## Run go vet
	go vet ./...

install: build ## Install binary to $GOPATH/bin or /usr/local/bin
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p ~/.local/bin || mkdir -p /usr/local/bin
	@if [ -w /usr/local/bin ]; then \
		cp bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME); \
		echo "Installed to /usr/local/bin/$(BINARY_NAME)"; \
	else \
		cp bin/$(BINARY_NAME) ~/.local/bin/$(BINARY_NAME); \
		echo "Installed to ~/.local/bin/$(BINARY_NAME)"; \
	fi

run: build ## Build and run the binary
	./bin/$(BINARY_NAME)

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Cleaned build artifacts"

deps: ## Download dependencies
	go mod download
	go mod tidy

deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

fmt-check: ## Check if code is formatted (does not modify files)
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "Error: Code is not formatted. Run 'make fmt' to fix."; \
		gofmt -d .; \
		exit 1; \
	fi
	@if command -v goimports > /dev/null 2>&1; then \
		if [ -n "$$(goimports -l -local github.com/unravelling/kilt .)" ]; then \
			echo "Error: Imports are not formatted. Run 'make fmt' to fix."; \
			goimports -d -local github.com/unravelling/kilt .; \
			exit 1; \
		fi; \
	fi
	@echo "Code formatting OK"

check: fmt-check vet lint test ## Run all checks (fmt-check, vet, lint, test)

.DEFAULT_GOAL := help



