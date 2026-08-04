# tuti-server

Go backend for the Tuti math tutor app. Provides:

- `POST /v1/chat` — send a message (with optional prior history and an
  optional attached capture) to the tutoring agent; streams the reply back
  as raw text chunks over a chunked HTTP response.
- `POST /v1/captures` — upload a screenshot (`multipart/form-data`, field
  `file`).
- `GET /v1/captures` — list uploaded captures, most recent first.
- `GET /v1/captures/{id}/content` — fetch a capture's raw bytes.
- `GET /healthz`

No authentication yet — every route is open. See `internal/httpapi/server.go`
for where an auth middleware should be added later.

## Design

Three seams are abstracted behind interfaces so the concrete backend can be
swapped without touching HTTP handlers:

| Interface                    | Default implementation                          |
| ----------------------------- | ------------------------------------------------ |
| `internal/agent.Agent`        | `internal/agent/claude` — streams from the Claude API |
| `internal/storage.Store`      | `internal/storage/localfs` — local disk, in-memory index |
| `internal/tracing.Tracer`     | `internal/tracing/slogtracer` — structured logs via `log/slog` |

To swap storage for S3/GCS or tracing for OpenTelemetry, implement the
respective interface and wire it in `cmd/server/main.go` — nothing else
changes.

## Running

```sh
cp .env.example .env   # then fill in ANTHROPIC_API_KEY
make run                # or: ./run.sh
```

`run.sh` loads `.env` (creating it from `.env.example` on first run) and
starts the server with `go run ./cmd/server`. `make run` is just a thin
wrapper around it; other useful targets:

```sh
make build   # compile to bin/server
make test    # go test ./...
make tidy    # go mod tidy
make fmt     # gofmt -l -w .
```

Credentials resolve the same way as every Anthropic SDK: `ANTHROPIC_API_KEY`,
then `ANTHROPIC_AUTH_TOKEN`, then an `ant auth login` profile. If none of
those are set, chat requests fail with a clear error; captures still work.

## Manual testing

[`../tuti-tui`](../tuti-tui) is a terminal client for poking at a running
server by hand — chat, capture upload/attach, health checks — without
reaching for `curl`. Run it alongside this server with `make run` from that
folder.

## Chat wire format

```
POST /v1/chat
{
  "message": "I've taken a photo of my math problem, can you help?",
  "history": [{"role": "user" | "agent", "text": "..."}],
  "captureId": "cap_xxxxx"   // optional, from a prior /v1/captures upload
}
```

The response is `Content-Type: text/plain`, flushed chunk-by-chunk as the
model streams its reply — read the body incrementally and append each chunk,
mirroring `Agent.sendMessage`'s `Stream<String>` on the Flutter client.
