.PHONY: build run test clean docker

build:
	go build -v ./...

run:
	go run ./cmd/server/main.go

test:
	go test -v -race -coverprofile=coverage.out ./...

test-integration:
	go test -v ./tests/integration/...

clean:
	rm -rf build/
	find . -name "*.test" -delete

docker-build:
	docker build -t hjtpx/hjtpx:latest .

docker-run:
	docker run -p 8080:8080 hjtpx/hjtpx:latest

lint:
	golangci-lint run

format:
	go fmt ./...

vet:
	go vet ./...
