#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

mkdir -p .tmp/go-build .tmp/go-cache .tmp/go-modcache .tmp/dev .tmp/dev/bin .tmp/dev/air
export TMPDIR="$root_dir/.tmp/go-build"
export PATH="/usr/local/go/bin:$PATH"
if command -v go >/dev/null 2>&1; then
  export PATH="$(go env GOPATH)/bin:$PATH"
fi

# Install once for live-reload loops inside the devcontainer.
if ! command -v air >/dev/null 2>&1 && [[ ! -x /go/bin/air && ! -x /root/go/bin/air ]]; then
  echo "air is missing from this devcontainer image; rebuild the devcontainer." >&2
  exit 1
fi
