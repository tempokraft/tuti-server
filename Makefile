.PHONY: run build test tidy fmt

# Run the server locally, loading .env if present.
run:
	@./run.sh

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -l -w .
