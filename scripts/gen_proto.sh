#!/usr/bin/env bash
# Regenerates Go code from ../tuti/proto/tuti_service.proto (the client repo
# owns the .proto — it's the source of truth for the API contract).
#
# Prerequisites (one-time setup):
#   go install github.com/bufbuild/buf/cmd/buf@latest
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
#
# Make sure $(go env GOPATH)/bin is on your PATH.
#
# The generated files land in internal/genproto/. Commit them so CI /
# teammates can build without running buf.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/../tuti/proto"

# ── Preflight checks ──────────────────────────────────────────────────────────

if [[ ! -d "${PROTO_DIR}" ]]; then
  echo "error: proto dir not found at ${PROTO_DIR}" >&2
  echo "  tuti-server must be checked out next to tuti (../tuti relative to this repo)." >&2
  exit 1
fi

if ! command -v buf &>/dev/null; then
  echo "error: buf not found." >&2
  echo "  Run: go install github.com/bufbuild/buf/cmd/buf@latest" >&2
  exit 1
fi

if ! command -v protoc-gen-go &>/dev/null; then
  echo "error: protoc-gen-go not found." >&2
  echo "  Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest" >&2
  exit 1
fi

# ── Generate ──────────────────────────────────────────────────────────────────

cd "${ROOT}"
mkdir -p internal/genproto

buf generate "${PROTO_DIR}"

echo "Generated files:"
find internal/genproto -name '*.go' 2>/dev/null || echo "  (none — check for errors above)"
echo "Done."
