BINARY_AGENT=instvisor-agent
BINARY_ANALYZE=instvisor-analyze
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

.PHONY: all build build-all clean install uninstall test coverage docker docker-build docker-push release help

all: build ## Build agent binary

build: ## Build agent binary
	@echo "Building $(BINARY_AGENT)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_AGENT) ./cmd/agent

build-analyze: ## Build analyze tool
	@echo "Building $(BINARY_ANALYZE)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_ANALYZE) ./cmd/analyze

build-all: build build-analyze ## Build all binaries

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

install: build-all ## Install binaries and systemd service
	@echo "Installing..."
	@sudo ./scripts/install.sh

uninstall: ## Uninstall instvisor
	@echo "Uninstalling..."
	@sudo ./scripts/uninstall.sh

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race ./...

coverage: ## Generate coverage report
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed" && exit 1)
	@golangci-lint run --timeout=10m ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

vet: ## Run go vet
	@echo "Running go vet..."
	@go vet ./...

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t abhishekkarki/instvisor:$(VERSION) .
	@docker tag abhishekkarki/instvisor:$(VERSION) abhishekkarki/instvisor:latest

docker-push: docker-build ## Push Docker image
	@echo "Pushing Docker image..."
	@docker push abhishekkarki/instvisor:$(VERSION)
	@docker push abhishekkarki/instvisor:latest

docker-run: ## Run in Docker
	@docker-compose up -d

docker-stop: ## Stop Docker container
	@docker-compose down

run: build ## Run agent locally
	@sudo ./$(BUILD_DIR)/$(BINARY_AGENT) -config configs/agent.yaml

analyze: build-analyze ## Run analysis
	@sudo ./$(BUILD_DIR)/$(BINARY_ANALYZE) -db /var/lib/instvisor/metrics.db

dev: build-all ## Development mode
	@sudo ./$(BUILD_DIR)/$(BINARY_AGENT) -config configs/agent.yaml

release: ## Create a new release (use: make release VERSION=v1.0.0)
	@echo "Creating release $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help