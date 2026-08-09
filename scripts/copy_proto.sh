#!/usr/bin/env bash
# Copies the .proto file from the sibling tuti repo into proto/ for local reference.
# The original is the source of truth — do not edit the copy here.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/../tuti/proto"

if [[ ! -d "${PROTO_DIR}" ]]; then
  echo "error: proto dir not found at ${PROTO_DIR}" >&2
  echo "  tuti-server must be checked out next to tuti (../tuti relative to this repo)." >&2
  exit 1
fi

mkdir -p "${ROOT}/proto"

copy_file() {
  local src="${PROTO_DIR}/$1"
  local dst="${ROOT}/proto/$1"
  if [[ ! -f "${src}" ]]; then
    echo "warning: ${src} not found, skipping" >&2
    return
  fi
  cp "${src}" "${dst}"
  echo "Copied: ${src}"
  echo "    to: ${dst}"
}

copy_file "tuti_service.proto"
copy_file "snap_solve_flow.md"
