#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

mkdir -p .tmp/go-build .tmp/go-cache .tmp/dev .tmp/dev/bin .tmp/dev/air
export TMPDIR="$root_dir/.tmp/go-build"
export HOME="${HOME:-/home/vscode}"
export PATH="/usr/local/go/bin:$PATH"
export GOPATH="${GOPATH:-/workspace/.tmp/go}"
export GOMODCACHE="${GOMODCACHE:-/tmp/go-modcache}"
export GOCACHE="${GOCACHE:-$root_dir/.tmp/go-cache}"
case " ${GOFLAGS:-} " in
*" -modcacherw "*) ;;
*) export GOFLAGS="${GOFLAGS:+${GOFLAGS} }-modcacherw" ;;
esac
mkdir -p "$GOCACHE" "$GOMODCACHE" "${GOPATH%/}/bin"
export PATH="$(go env GOPATH)/bin:$PATH"

# Install once for live-reload loops inside the devcontainer.
if ! command -v air >/dev/null 2>&1 && [[ ! -x /go/bin/air && ! -x /root/go/bin/air ]]; then
  echo "air is missing from this devcontainer image; rebuild the devcontainer." >&2
  exit 1
fi
