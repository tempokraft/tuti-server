.PHONY: run build test tidy fmt proto

# Run the server locally, loading .env if present.
run:
	@./run.sh

build:
	go build -o bin/server ./cmd/server

# Regenerate Go bindings from ../tuti/proto/tuti_service.proto.
proto:
	@./scripts/gen_proto.sh

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -l -w .
