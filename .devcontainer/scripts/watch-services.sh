#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"

mkdir -p .tmp/go-build .tmp/go-cache .tmp/dev .tmp/dev/bin .tmp/dev/air
export TMPDIR="$root_dir/.tmp/go-build"
export HOME="${HOME:-/home/vscode}"
export GOPATH="${GOPATH:-/workspace/.tmp/go}"
export GOMODCACHE="${GOMODCACHE:-/tmp/go-modcache}"
export GOCACHE="${GOCACHE:-$root_dir/.tmp/go-cache}"
case " ${GOFLAGS:-} " in
*" -modcacherw "*) ;;
*) export GOFLAGS="${GOFLAGS:+${GOFLAGS} }-modcacherw" ;;
esac
export PATH="/usr/local/go/bin:/go/bin:/root/go/bin:${GOPATH%/}/bin:$PATH"
mkdir -p "$GOCACHE" "$GOMODCACHE" "${GOPATH%/}/bin"

if ! command -v air >/dev/null 2>&1 && [[ ! -x /go/bin/air && ! -x /root/go/bin/air && ! -x "${GOPATH%/}/bin/air" ]]; then
  echo "air is not installed in the devcontainer image; rebuild with make up" >&2
  exit 1
fi

env_file=".env"
if [[ ! -f "$env_file" ]]; then
  cp "${ENV_EXAMPLE:-.env.local.example}" "$env_file"
fi

set -a
. "$env_file"
set +a

if [[ -z "${FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY:-}" || -z "${FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY:-}" ]]; then
  eval "$(go run ./cmd/join-grant-key)"
  export FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY
  export FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY
fi

export FRACTURING_SPACE_JOIN_GRANT_ISSUER="${FRACTURING_SPACE_JOIN_GRANT_ISSUER:-fracturing.space/auth}"
export FRACTURING_SPACE_JOIN_GRANT_AUDIENCE="${FRACTURING_SPACE_JOIN_GRANT_AUDIENCE:-fracturing.space/game}"
export FRACTURING_SPACE_JOIN_GRANT_TTL="${FRACTURING_SPACE_JOIN_GRANT_TTL:-5m}"
export FRACTURING_SPACE_GAME_EVENT_HMAC_KEY="${FRACTURING_SPACE_GAME_EVENT_HMAC_KEY:-dev-secret}"
export FRACTURING_SPACE_AUTH_HTTP_ADDR="${FRACTURING_SPACE_AUTH_HTTP_ADDR:-0.0.0.0:8084}"
export FRACTURING_SPACE_WEB_HTTP_ADDR="${FRACTURING_SPACE_WEB_HTTP_ADDR:-0.0.0.0:8086}"
export FRACTURING_SPACE_MCP_TRANSPORT="${FRACTURING_SPACE_MCP_TRANSPORT:-http}"
export FRACTURING_SPACE_MCP_HTTP_ADDR="${FRACTURING_SPACE_MCP_HTTP_ADDR:-0.0.0.0:8081}"

pids=()

start_service() {
  local name="$1"
  : > ".tmp/dev/${name}.log"
  air -c ".devcontainer/air/${name}.toml" >> ".tmp/dev/${name}.log" 2>&1 &
  pids+=("$!")
  printf 'started %s watcher (pid %s)\n' "$name" "$!"
}

cleanup() {
  trap - EXIT INT TERM
  for pid in "${pids[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done
  wait || true
  rm -f .tmp/dev/watch-services.pid
}

trap cleanup EXIT INT TERM

start_service game
start_service auth
start_service mcp
start_service admin
start_service web

wait -n "${pids[@]}"
exit $?
