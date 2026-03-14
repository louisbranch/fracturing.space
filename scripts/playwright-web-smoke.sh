#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

export BASE_URL="${BASE_URL:-http://localhost:8080}"
export ARTIFACT_ROOT="${ARTIFACT_ROOT:-artifacts/playwright}"
export WEB_SMOKE_REQUIRE_AUTH="${WEB_SMOKE_REQUIRE_AUTH:-1}"

if [[ -n "${WEB_SMOKE_AUTH_ADDR:-}" && -z "${WEB_SMOKE_SESSION_ID:-}" ]]; then
  session_env="$(go run ./internal/tools/websmokeauth \
    -auth-addr "${WEB_SMOKE_AUTH_ADDR}" \
    -ttl-seconds "${WEB_SMOKE_SESSION_TTL_SECONDS:-3600}" \
    -username "${WEB_SMOKE_AUTH_USERNAME:-}" \
    -recipient-username "${WEB_SMOKE_AUTH_RECIPIENT_USERNAME:-}")"
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    export "$line"
  done <<< "$session_env"
fi

if [[ "${WEB_SMOKE_REQUIRE_AUTH}" == "1" && -z "${WEB_SMOKE_SESSION_ID:-}" ]]; then
  cat >&2 <<'EOF'
web smoke requires authenticated coverage, but WEB_SMOKE_SESSION_ID is empty.
Set WEB_SMOKE_AUTH_ADDR with WEB_SMOKE_AUTH_USERNAME and WEB_SMOKE_AUTH_RECIPIENT_USERNAME to mint a session for existing accounts, or set WEB_SMOKE_SESSION_ID directly.
Set WEB_SMOKE_REQUIRE_AUTH=0 to intentionally run unauthenticated-only smoke coverage.
EOF
  exit 1
fi

if [[ "${WEB_SMOKE_REQUIRE_AUTH}" == "1" ]] && [[ -z "${WEB_SMOKE_RECIPIENT_USER_ID:-}" ]] && [[ -z "${WEB_SMOKE_USER_ID:-}" ]]; then
  cat >&2 <<'EOF'
web smoke requires invite recipient identity for deterministic mutation checks.
Set WEB_SMOKE_AUTH_ADDR with WEB_SMOKE_AUTH_USERNAME and WEB_SMOKE_AUTH_RECIPIENT_USERNAME, or set WEB_SMOKE_RECIPIENT_USER_ID (or legacy WEB_SMOKE_USER_ID) directly.
EOF
  exit 1
fi

exec "$ROOT/scripts/playwright-run-spec.sh" "$ROOT/docs/specs/web-smoke.md"
