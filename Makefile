.PHONY: build build-linux build-arm64 run dev test clean install uninstall

# Build variables
BINARY_NAME=hivedeck-agent
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for Linux (amd64)
build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .

# Build for Linux (arm64) - Raspberry Pi
build-arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .

# Run the agent
run: build
	./$(BINARY_NAME)

# Run with hot reload (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/cosmtrek/air@latest)
	air

# Run tests
test:
	go test -v -race -cover ./...

# Run tests with coverage report
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-linux-* coverage.out coverage.html

# Install on current system (requires root)
install:
	@./scripts/install.sh

# Uninstall from current system (requires root)
uninstall:
	@./scripts/uninstall.sh

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

# Download dependencies
deps:
	go mod download
	go mod tidy

# Build and deploy to Pi
deploy-pi: build-arm64
	scp $(BINARY_NAME)-linux-arm64 root@pi:/tmp/$(BINARY_NAME)
	ssh root@pi 'systemctl stop $(BINARY_NAME) || true && mv /tmp/$(BINARY_NAME) /home/ngenoh/dev/backend/$(BINARY_NAME)/$(BINARY_NAME) && chmod +x /home/ngenoh/dev/backend/$(BINARY_NAME)/$(BINARY_NAME) && systemctl start $(BINARY_NAME)'
	@echo "Deployed to Pi"

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-linux - Build for Linux (amd64)"
	@echo "  build-arm64 - Build for Linux (arm64/Raspberry Pi)"
	@echo "  run         - Build and run"
	@echo "  dev         - Run with hot reload"
	@echo "  test        - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean       - Clean build artifacts"
	@echo "  install     - Install systemd service"
	@echo "  uninstall   - Uninstall systemd service"
	@echo "  fmt         - Format code"
	@echo "  lint        - Lint code"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  deploy-pi   - Build and deploy to Raspberry Pi"
