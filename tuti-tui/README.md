# tuti-tui

An interactive terminal client for exercising a running
[tuti-server](..) instance by hand: chat with the tutoring
agent, upload and attach captures, and check server health — all from the
terminal, no `curl` required.

## Running

Start `tuti-server` first (see its README), then in this folder:

```sh
make run                      # connects to http://localhost:8080
make run SERVER=http://localhost:9090   # or a different host/port
```

Equivalently:

```sh
go run ./cmd/tuti-tui -server http://localhost:8080
# or
export TUTI_SERVER_URL=http://localhost:8080
go run ./cmd/tuti-tui
```

Other targets:

```sh
make build   # compile to bin/tuti-tui
make test    # go test ./...
make tidy    # go mod tidy
make fmt     # gofmt -l -w .
```

## Using it

- Type a message and press **Enter** to send it; the reply streams in as
  it's generated.
- **Ctrl+U** prompts for a local file path and uploads it as a capture.
  Once uploaded, it's automatically attached to your next message.
- **Ctrl+L** lists previously uploaded captures — pick one with the arrow
  keys and **Enter** to attach it instead.
- **Esc** detaches the currently-attached capture (when at the chat
  prompt), or cancels an upload/picker prompt.
- **Ctrl+R** rechecks server health (also shown live in the status bar).
- **Ctrl+C** quits.

Conversation history is kept in memory and sent with each request, mirroring
how the Flutter client drives `POST /v1/chat`.
