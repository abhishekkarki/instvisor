BINARY_NAME=instvisor-agent
VERSION=0.1.0
BUILD_DIR=build

.PHONY: all build clean install test run

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/agent

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f /tmp/instvisor-test.db

install: build
	@echo "Installing $(BINARY_NAME)..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo mkdir -p /etc/instvisor
	sudo cp configs/agent.yaml /etc/instvisor/
	sudo mkdir -p /var/lib/instvisor
	@echo "Installation complete!"

test:
	go test -v ./...

run: build
	@echo "Running agent with test config..."
	./$(BUILD_DIR)/$(BINARY_NAME) -config configs/agent.yaml

dev: build
	@echo "Running in development mode..."
	./$(BUILD_DIR)/$(BINARY_NAME) -config configs/agent.yaml