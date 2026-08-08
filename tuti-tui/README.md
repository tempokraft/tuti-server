# tuti-tui

An interactive terminal client for exercising a running
[tuti-server](..) instance by hand: every `TutiService` RPC (see
[`tuti/proto/tuti_service.proto`](../../tuti/proto/tuti_service.proto)),
driven from a single command line — no `curl` required.

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

Type a command and press **Enter**. `help` lists them all; the short
version:

```
health                          recheck server health
init                            start a Snap & Solve session
snap <path>                     upload a photo for the current snap session
respond <check_work|solve|explain>
                                 submit the student's chosen action
upload <path>                   upload a screenshot (standalone)
captures                        list uploaded captures
session                         create a solve session
analyze [id1,id2,...]           analyze captures (defaults to the last upload)
lesson <id> [lang]              fetch lesson content (lang defaults to en)
clear                           clear the transcript
quit / ctrl+c                   exit
```

A typical Snap & Solve run:

```
init
snap ./page.jpg
respond check_work
```

The status bar tracks context between commands — the current snap session
id, solve session id, and last uploaded capture id — so `analyze` and
`respond` work without re-typing ids by hand. Every result is shown as a
short human-readable summary followed by the full response body (long
strings, like base64 image bytes, are truncated for readability).

`respond` and `analyze` call out to the server's configured LLM backend and
can take a while — everything else responds quickly.
