.PHONY: run build dev test tidy fmt proto

# Run the server locally via go run (no prior build required).
run:
	@./run.sh

build:
	@./scripts/copy_proto.sh
	go build -o bin/server ./cmd/server

# Build then run the compiled binary (mirrors production more closely than `run`).
dev: build
	@./run.sh bin/server

# Regenerate Go bindings from ../tuti/proto/tuti_service.proto.
proto:
	@./scripts/gen_proto.sh

test:
	go test ./...

tidy:
	go mod tidy

fmt:
	gofmt -l -w .
