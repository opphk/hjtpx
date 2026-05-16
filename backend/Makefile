.PHONY: test test-cover test-unit test-integration clean lint vet fmt

test:
	go test ./... -v

test-cover:
	go test ./... -v -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-cover-ci:
	go test ./... -v -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

test-unit:
	go test -v -run "^Test[A-Z]" ./...

test-integration:
	go test -v -run "^Test.*Integration$" ./...

test-race:
	go test -race -v ./...

test-all:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -f coverage.out coverage.html

lint:
	go vet ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

deps:
	go mod download
	go mod tidy

build:
	go build ./...

run:
	go run ./cmd/api/main.go

coverage-report:
	@echo "=== Coverage Summary ==="
	@go tool cover -func=coverage.out | tail -n 1
	@echo ""
	@echo "=== Detailed Coverage ==="
	@go tool cover -func=coverage.out
