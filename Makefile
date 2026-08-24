.PHONY: all build test lint clean run

BINARY_NAME=code-reviewer

all: test build

build:
	go build -o bin/$(BINARY_NAME) .

test:
	go test -v -race ./...

lint:
	golangci-lint run ./... || go vet ./...

clean:
	rm -rf bin/ coverage.out

run:
	go run . --workspace . --diff
