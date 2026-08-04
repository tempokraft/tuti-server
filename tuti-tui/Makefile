.PHONY: run build test tidy fmt

# Run the TUI against a local tuti-server (override with SERVER=...).
SERVER ?= http://localhost:8080

run:
	go run ./cmd/tuti-tui -server $(SERVER)

build:
	go build -o bin/tuti-tui ./cmd/tuti-tui

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -l -w .
