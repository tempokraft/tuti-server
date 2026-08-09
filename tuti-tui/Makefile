.PHONY: run build dev test tidy fmt

# Run the TUI against a local tuti-server (override with SERVER=...).
SERVER ?= http://localhost:8080

run:
	go run ./cmd/tuti-tui -server $(SERVER)

build:
	go build -o bin/tuti-tui ./cmd/tuti-tui

# Build then run the compiled binary.
dev: build
	./bin/tuti-tui -server $(SERVER)

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -l -w .
