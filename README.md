# tuti-server

Go backend for the Tuti math tutor app: an HTTP/JSON binding of the
`TutiService` RPCs defined in
[`tuti/proto/tuti_service.proto`](../tuti/proto/tuti_service.proto) — the
**source of truth** for the API contract, owned by the client repo. This
server implements it; it doesn't define it. See that file for the
authoritative message/RPC shapes.

Routes (all `POST`, protojson request/response bodies — see
[Wire format](#wire-format) below):

- `POST /v1/InitializeSnapAndSolve` — start a Snap & Solve session.
- `POST /v1/SubmitSnap` — upload the captured photo for a session.
- `POST /v1/SubmitSnapResponse` — submit the student's chosen action
  (`check_work` / `solve` / `explain`); returns the analysis.
- `POST /v1/UploadScreenshot` — upload a screenshot.
- `POST /v1/ListCaptures` — list uploaded captures, most recent first.
- `POST /v1/CreateSession` — open a solve session.
- `POST /v1/AnalyzeAssets` — analyze captures attached to a session.
- `POST /v1/GetLessonContent` — fetch lesson content by id + language.
- `GET /healthz`

No authentication yet — every route is open. See `internal/httpapi/server.go`
for where an auth middleware should be added later.

## Design

| Package                  | Role                                                                 |
| ------------------------- | --------------------------------------------------------------------- |
| `internal/genproto/tutiv1` | Generated Go types from the proto — see [Regenerating](#regenerating-the-proto-bindings) |
| `internal/catalog`        | Static content: lessons, practice problems, topic recommendations, snap options. Ported from the Flutter app's `MockTutiServerClient`. |
| `internal/analysis`       | The one place a photo is actually sent to a vision LLM: classifying blank-vs-written and extracting/evaluating a problem (forced tool-use, not free text). Backend is pluggable — see `ANALYSIS_BACKEND` below. Everything else (similar problems, lessons to review) is resolved deterministically from `internal/catalog`. |
| `internal/session`        | `Store` interface + `internal/session/filestore` — local disk, in-memory index, for Snap & Solve step-flow and solve-session state. |
| `internal/storage`        | `Store` interface + `internal/storage/localfs` — local disk, in-memory index, for captures. |
| `internal/tracing`        | `Tracer` interface + `internal/tracing/slogtracer` — structured logs via `log/slog`. |

To swap session/capture storage for a database or object store, or
tracing for OpenTelemetry, implement the respective interface and wire it
in `cmd/server/main.go` — nothing else changes.

## Running

```sh
cp .env.example .env   # then fill in OPENAI_API_KEY (or switch to Anthropic, see below)
make run                # or: ./run.sh
```

`run.sh` loads `.env` (creating it from `.env.example` on first run) and
starts the server with `go run ./cmd/server`. `make run` is just a thin
wrapper around it; other useful targets:

```sh
make build   # compile to bin/server
make proto   # regenerate internal/genproto from ../tuti/proto
make test    # go test ./...
make tidy    # go mod tidy
make fmt     # gofmt -l -w .
```

`internal/analysis` supports two backends, selected by `ANALYSIS_BACKEND`
(`openai`, the default, or `anthropic`) — see `internal/analysis/provider.go`
for the interface if you want to add a third. `ANALYSIS_MODEL` picks the
model for whichever backend is active (defaults: `gpt-5.2` /
`claude-opus-5`). Credentials resolve the same way as the respective vendor
SDK: for OpenAI, `OPENAI_API_KEY`; for Anthropic, `ANTHROPIC_API_KEY`, then
`ANTHROPIC_AUTH_TOKEN`, then an `ant auth login` profile. The server logs
which backend/model it resolved and whether a key was found for it at
startup — check those logs first if analysis calls are failing. If no
credentials are set, `SubmitSnapResponse`/`AnalyzeAssets` fail once they
actually call the model; everything else (captures, lessons, session
bookkeeping) still works.

## Regenerating the proto bindings

The `.proto` lives in the client repo, `tuti/proto/tuti_service.proto` —
edit it there. This repo only generates Go bindings from it:

```sh
make proto   # or: ./scripts/gen_proto.sh
```

Requires `tuti-server` checked out next to `tuti` (`../tuti` from this
repo's root), plus `buf` and `protoc-gen-go` on `PATH`:

```sh
go install github.com/bufbuild/buf/cmd/buf@latest
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

Generated output lands in `internal/genproto/tutiv1/` and is committed, so
building this repo doesn't require `buf` unless the proto actually changed.

## Manual testing

[`tuti-tui`](tuti-tui) is a terminal client for poking at a running server
by hand — every RPC above, driven from a single command line.

## Wire format

Requests/responses are [protojson](https://protobuf.dev/programming-guides/proto3/#json)
encodings of the proto messages: fields are `camelCase`, `int64` fields
(e.g. `uploadedAtMs`) are JSON strings, `bytes` fields (e.g. `data` on
`Capture`/`UploadScreenshotRequest`/`SubmitSnapRequest`) are base64, and a
`oneof` (e.g. `NextStep.step`, `ContentBlock.block`) appears as a plain
object with exactly one of its variant fields set — e.g.
`{"nextStep": {"captureSnap": {}}}` or `{"block": {"math": {"expression": "x^2"}}}`.
Empty-request RPCs (`ListCaptures`, `CreateSession`) accept an empty body.

Example — the Snap & Solve flow end to end:

```
POST /v1/InitializeSnapAndSolve
{}
→ {"sessionId": "snap_...", "nextStep": {"captureSnap": {}}}

POST /v1/SubmitSnap
{"sessionId": "snap_...", "filename": "page.jpg", "data": "<base64>"}
→ {"nextStep": {"captureSnapResponse": {"options": [...]}}}

POST /v1/SubmitSnapResponse
{"sessionId": "snap_...", "responseId": "check_work"}
→ {"nextStep": {"displayAnalysis": {"lessonsToReview": [...], "problemsCaptured": [...]}}}
```
