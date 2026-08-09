# tuti-tui

An interactive terminal client for exercising a running
[tuti-server](..) instance by hand: every `TutiService` RPC (see
[`tuti/proto/tuti_service.proto`](../../tuti/proto/tuti_service.proto)) —
**commands on the left, parameters on the right**, no `curl` required.

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

The screen is two panes above a results log:

- **Left — Commands**: every RPC, plus `Clear Log`. `↑`/`↓` to pick one.
- **Right — Parameters**: the fields the selected command needs (empty for
  RPCs like `Health Check` or `List Captures`).

Press **Enter** on a command with no parameters to run it immediately;
otherwise Enter moves you into its parameter form.

In the form:

| Key | Does |
| --- | --- |
| `Tab` / `Shift+Tab` | move between fields (past the first/last returns to the menu) |
| `←` / `→` | cycle a choice field (e.g. `Submit Response`'s check_work/solve/explain) |
| `Ctrl+F` | **browse your machine for a file** — opens a file picker for the focused field (photo parameters only) |
| `Enter` | on a file field, opens the file picker; otherwise runs the command |
| `Ctrl+S` | always runs the command, regardless of which field is focused |
| `Esc` | back to the menu, keeping whatever you've typed |

In the file picker: arrow keys navigate, `Enter` opens a directory or
selects a file, `Esc` cancels back to the form without changing the field.

A typical Snap & Solve run: select **Init Snap & Solve** and press Enter,
then **Submit Snap**, `Ctrl+F` to pick a photo from disk, `Ctrl+S` to
submit, then **Submit Response**, pick an option with `←`/`→`, `Ctrl+S`.

The status bar tracks context between commands — the current snap session
id, solve session id, and last uploaded capture id — so `Analyze Assets`
and `Submit Response` work without re-typing ids by hand. Every result
shows a short human-readable summary followed by the full response body
(long strings, like base64 image bytes, are truncated for readability).

`Submit Response` and `Analyze Assets` call out to the server's configured
LLM backend and can take a while — everything else responds quickly.
