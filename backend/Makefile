.PHONY: test test-cover test-unit test-integration test-fuzz test-chaos test-security test-all-v3 clean lint vet fmt

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

# New test types for v19.0
test-fuzz:
	go test -v ./internal/testing/fuzz/...
	go test -v ./internal/service -run TestCaptchaFuzzing

test-chaos:
	go test -v ./internal/testing/chaos/... -run RunChaosExperiments

test-security:
	go test -v ./internal/testing/security/... -run RunOWASPSecurityTests

test-all-v3:
	@echo "=== Running All Tests (v19.0) ==="
	@echo "1. Running Unit Tests..."
	go test -v -run "^Test[A-Z]" ./...
	@echo ""
	@echo "2. Running Fuzz Tests..."
	go test -v ./internal/testing/fuzz/...
	go test -v ./internal/service -run TestCaptchaFuzzing
	@echo ""
	@echo "3. Running Chaos Engineering Tests..."
	go test -v ./internal/testing/chaos/... -run RunChaosExperiments
	@echo ""
	@echo "4. Running Security Tests (OWASP Top 10)..."
	go test -v ./internal/testing/security/... -run RunOWASPSecurityTests
	@echo ""
	@echo "=== All Tests Completed ==="

test-all:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

clean:
	rm -f coverage.out coverage.html fuzz-*.log

lint:
	golangci-lint run

lint-fast:
	golangci-lint run --fast

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
