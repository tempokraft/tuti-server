# CLAUDE.md — tuti-server

Go HTTP backend for the Tuti math tutoring app. Connects a mobile client to vision-LLM analysis pipelines.

---

## Architecture Overview

```
cmd/server/main.go          Entry point, wiring
internal/
  httpapi/                  HTTP handlers, routing, middleware
  session/                  In-memory + durable session state
    filestore/              JSON file-backed session persistence
  analysis/                 Vision-LLM orchestration
  catalog/                  Static lesson/problem/topic content
  storage/localfs/          Capture (image) persistence
  tracing/slogtracer/       Structured logging via slog
  config/                   Env-var configuration
  genproto/tutiv1/          Generated proto bindings (committed, do not edit)
```

**Key principle:** the HTTP layer is a thin shell — it only marshals/unmarshals. Business logic lives in `session`, `analysis`, and `catalog`.

**No web framework.** Uses Go's stdlib `net/http` with `http.ServeMux`.

---

## Data Flow

### Snap & Solve (multi-step)

```
InitializeSnapAndSolve → SubmitSnap → SubmitSnapResponse
```

Session tracked by `SnapStore` as a state machine:
- `stepAwaitSnap` → `stepAwaitResponse` → `stepDone`
- Session IDs: `snap_<16 hex chars>`
- Out-of-order calls return `ErrWrongStep` → HTTP 409

### Solve Session (lightweight)

```
CreateSession → AnalyzeAssets
```

Session IDs: `sess_<16 hex chars>`. Session only validates existence; asset IDs passed per-call.

---

## ID Prefixes

| Prefix | Domain |
|--------|--------|
| `cap_` | Uploaded captures (images) |
| `snap_` | Snap & Solve sessions |
| `sess_` | Solve sessions |
| `detected_` | AI-extracted problems |

All IDs are `<prefix><16 random hex chars>`.

---

## Session Persistence

Write-through pattern: always write to filestore before updating in-memory index.

```go
// Correct order:
if err := s.store.Save(ctx, session); err != nil { return err }
s.index[id] = session  // only after durable write
```

On startup, `SnapStore`/`SolveStore` rehydrate their in-memory index from the filestore. Disk is the source of truth. Use `sync.Mutex` for all in-memory index access.

---

## Analysis Backends

`Analyzer` is an interface in `internal/analysis/analysis.go`. Two implementations:

- `provider_anthropic.go` — Claude (Anthropic SDK)
- `provider_openai.go` — OpenAI

To add a new backend: implement the `provider` interface, add a case in `main.go`. Nothing else changes.

Active backend is set via `ANALYSIS_BACKEND` env var (`"anthropic"` or `"openai"`).

---

## HTTP Layer Conventions

**Middleware stack** (applied in `server.go`):

```
withTracing → withRecover → withCORS → handler
```

**Handler function naming:** `handle<OperationName>` (e.g., `handleSubmitSnap`).

**Request/response helpers** (all in `httpapi/proto.go`):

```go
readProto(r, &req)          // unmarshal protojson body
writeProto(w, &resp)        // marshal protojson response
writeJSONError(w, code, msg) // {"error": "..."} + status
```

**Wire format:** protojson (camelCase fields, int64 as JSON string, bytes as base64).

**Error → HTTP status mapping:**

| Error | Status |
|-------|--------|
| `session.ErrNotFound` / `storage.ErrNotFound` | 404 |
| `session.ErrWrongStep` | 409 |
| Analysis failure (external LLM) | 502 |
| Validation / bad request | 400 |

Always call `span.RecordError(err)` before returning an error response.

---

## CORS

Wildcard CORS is enabled for all origins. Methods: `GET, POST, OPTIONS`. Headers: `Content-Type, Authorization`. Preflight OPTIONS requests are handled automatically by `withCORS`.

---

## Configuration

All configuration comes from environment variables. See `internal/config/config.go`.

Key vars:

| Var | Default | Notes |
|-----|---------|-------|
| `PORT` | `8080` | Listen port |
| `STORAGE_DIR` | `./data/captures` | Image upload dir |
| `SESSION_DIR` | `./data/sessions` | Session persistence dir |
| `ANALYSIS_BACKEND` | `openai` | `openai` or `anthropic` |
| `ANALYSIS_MODEL` | model per backend | Override LLM model |
| `ANTHROPIC_API_KEY` | — | Falls back to SDK resolution |
| `OPENAI_API_KEY` | — | Falls back to SDK resolution |
| `MAX_UPLOAD_BYTES` | `10485760` (10 MiB) | Capture upload limit |

Add new config fields with a `getEnv()` / `getEnvInt64()` helper and a `default*` constant.

---

## Naming Conventions

| Pattern | Used for |
|---------|----------|
| `handle*` | HTTP handler functions |
| `write*` | HTTP response writers |
| `read*` | HTTP request body readers |
| `build*` | Complex struct constructors |
| `step*` | Session state constants |
| `default*` | Config default values |
| `*Kind`, `*Prefix` | Domain constants |

Interfaces use `-er`/`-or` suffix: `Analyzer`, `Tracer`, `Store`. Receivers are single letters (`s`, `t`, `p`).

---

## Proto Bindings

Generated bindings live in `internal/genproto/tutiv1/`. **Never edit these manually.**

To regenerate after proto changes:

```bash
make proto
# or
./scripts/gen_proto.sh
```

Requires `buf` and `protoc-gen-go` on PATH. Proto source of truth is in the sibling `tuti/proto` repo.

---

## Testing

- Integration-style: tests for `session` use real filestore with `t.TempDir()`, not mocks. This verifies rehydration across simulated restarts.
- Analysis tests use a `mockProvider` (implements `provider` interface) to avoid real API calls.
- Test files live alongside the code they test (`package foo_test` or `package foo`).
- Cover error paths (`ErrNotFound`, `ErrWrongStep`) not just happy paths.

---

## Logging / Tracing

Structured JSON logs via `log/slog`. Every HTTP request is wrapped in a span by `withTracing` middleware. Spans emit: method, path, status code, duration, and any recorded errors.

Use `span.RecordError(err)` — not bare `log.Printf` — when an error occurs inside a handler.

---

## Graceful Shutdown

Server listens for `SIGINT`/`SIGTERM`. On signal, `Shutdown(ctx)` is called with a 10-second timeout. Handlers should respect context cancellation.
