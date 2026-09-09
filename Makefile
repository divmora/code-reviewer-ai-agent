.PHONY: build clean test test-coverage fmt lint docker-build docker-build-multiarch run

BINARY_NAME ?= code-reviewer
IMAGE_NAME ?= ghcr.io/divmora/code-reviewer-ai-agent

build:
	@go build -o bin/$(BINARY_NAME) .

clean:
	@rm -rf bin/ coverage.out coverage.html

test:
	@go test -v -race ./...

test-coverage:
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html

fmt:
	@go fmt ./...

lint:
	@golangci-lint run ./... || go vet ./...

docker-build:
	@docker build -t $(IMAGE_NAME):latest .

docker-build-multiarch:
	@docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE_NAME):latest .

run:
	@go run . --workspace . --diff
