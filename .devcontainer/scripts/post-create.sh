#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

mkdir -p .tmp/go-build .tmp/go-cache .tmp/go-modcache .tmp/dev .tmp/dev/bin .tmp/dev/air
export TMPDIR="$root_dir/.tmp/go-build"
export GOTMPDIR="$root_dir/.tmp/go-build"
export GOCACHE="$root_dir/.tmp/go-cache"
export GOMODCACHE="$root_dir/.tmp/go-modcache"

# Install once for live-reload loops inside the devcontainer.
if ! command -v air >/dev/null 2>&1; then
  CGO_ENABLED=0 go install github.com/air-verse/air@v1.62.0
fi
