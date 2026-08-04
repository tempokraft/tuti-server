#!/usr/bin/env bash
# Runs the tuti-server locally, loading config from .env if present.
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"

if [ ! -f .env ]; then
	if [ -f .env.example ]; then
		echo "No .env found — creating one from .env.example."
		cp .env.example .env
		echo "Edit .env and set ANTHROPIC_API_KEY, then re-run this script."
		exit 1
	else
		echo "No .env or .env.example found; running with defaults + inherited environment."
	fi
fi

if [ -f .env ]; then
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
fi

exec go run ./cmd/server
