.PHONY: build run test lint clean fmt deps

BINARY_NAME=captcha-system
BUILD_DIR=./bin
MAIN_PATH=./cmd/server

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

run: build
	@echo "Running..."
	@$(BUILD_DIR)/$(BINARY_NAME)

test:
	@echo "Running tests..."
	go test -v -cover ./...

lint:
	@echo "Running linter..."
	go vet ./...
	golangci-lint run

fmt:
	@echo "Formatting code..."
	go fmt ./...

deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

mod:
	@echo "Initializing Go module..."
	go mod init github.com/opphk/captcha-system

docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .

docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 $(BINARY_NAME):latest
