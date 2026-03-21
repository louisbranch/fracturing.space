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

load_or_generate_ai_encryption_key() {
  local key_file=".tmp/dev/ai-encryption.key"
  local key_value="${FRACTURING_SPACE_AI_ENCRYPTION_KEY:-}"

  if [[ -z "$key_value" && -s "$key_file" ]]; then
    key_value="$(tr -d '\r\n' < "$key_file")"
  fi
  if [[ -z "$key_value" ]]; then
    key_value="$(dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\r\n')"
  fi
  if [[ -z "$key_value" ]]; then
    echo "failed to initialize FRACTURING_SPACE_AI_ENCRYPTION_KEY" >&2
    exit 1
  fi

  printf '%s' "$key_value" > "$key_file"
  chmod 600 "$key_file"
  export FRACTURING_SPACE_AI_ENCRYPTION_KEY="$key_value"
}

load_or_generate_ai_encryption_key

export FRACTURING_SPACE_JOIN_GRANT_ISSUER="${FRACTURING_SPACE_JOIN_GRANT_ISSUER:-fracturing.space/auth}"
export FRACTURING_SPACE_JOIN_GRANT_AUDIENCE="${FRACTURING_SPACE_JOIN_GRANT_AUDIENCE:-fracturing.space/game}"
export FRACTURING_SPACE_JOIN_GRANT_TTL="${FRACTURING_SPACE_JOIN_GRANT_TTL:-5m}"
export FRACTURING_SPACE_PLAY_LAUNCH_GRANT_ISSUER="${FRACTURING_SPACE_PLAY_LAUNCH_GRANT_ISSUER:-fracturing-space-web}"
export FRACTURING_SPACE_PLAY_LAUNCH_GRANT_AUDIENCE="${FRACTURING_SPACE_PLAY_LAUNCH_GRANT_AUDIENCE:-fracturing-space-play}"
export FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY="${FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY:-MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=}"
export FRACTURING_SPACE_PLAY_LAUNCH_GRANT_TTL="${FRACTURING_SPACE_PLAY_LAUNCH_GRANT_TTL:-2m}"
export FRACTURING_SPACE_GAME_EVENT_HMAC_KEY="${FRACTURING_SPACE_GAME_EVENT_HMAC_KEY:-dev-secret}"
export FRACTURING_SPACE_GAME_PORT="${FRACTURING_SPACE_GAME_PORT:-8082}"
export FRACTURING_SPACE_GAME_ADDR="${FRACTURING_SPACE_GAME_ADDR:-localhost:8082}"
export FRACTURING_SPACE_GAME_CONTENT_DB_PATH="${FRACTURING_SPACE_GAME_CONTENT_DB_PATH:-data/game-content.db}"
export FRACTURING_SPACE_AUTH_ADDR="${FRACTURING_SPACE_AUTH_ADDR:-localhost:8083}"
export FRACTURING_SPACE_AUTH_HTTP_ADDR="${FRACTURING_SPACE_AUTH_HTTP_ADDR:-0.0.0.0:8084}"
export FRACTURING_SPACE_SOCIAL_PORT="${FRACTURING_SPACE_SOCIAL_PORT:-8090}"
export FRACTURING_SPACE_SOCIAL_ADDR="${FRACTURING_SPACE_SOCIAL_ADDR:-localhost:8090}"
export FRACTURING_SPACE_DISCOVERY_PORT="${FRACTURING_SPACE_DISCOVERY_PORT:-8091}"
export FRACTURING_SPACE_DISCOVERY_ADDR="${FRACTURING_SPACE_DISCOVERY_ADDR:-localhost:8091}"
export FRACTURING_SPACE_ADMIN_ADDR="${FRACTURING_SPACE_ADMIN_ADDR:-0.0.0.0:8081}"
export FRACTURING_SPACE_WEB_HTTP_ADDR="${FRACTURING_SPACE_WEB_HTTP_ADDR:-0.0.0.0:8080}"
export FRACTURING_SPACE_PLAY_HTTP_ADDR="${FRACTURING_SPACE_PLAY_HTTP_ADDR:-0.0.0.0:8094}"
export FRACTURING_SPACE_PLAY_DB_PATH="${FRACTURING_SPACE_PLAY_DB_PATH:-data/play.db}"
export FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL="${FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL:-}"
export FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS="${FRACTURING_SPACE_WEBAUTHN_RP_ORIGINS:-http://localhost:8080}"
export FRACTURING_SPACE_WEB_AUTH_ADDR="${FRACTURING_SPACE_WEB_AUTH_ADDR:-localhost:8083}"
export FRACTURING_SPACE_OAUTH_LOGIN_UI_URL="${FRACTURING_SPACE_OAUTH_LOGIN_UI_URL:-${FRACTURING_SPACE_PUBLIC_SCHEME:-http}://${FRACTURING_SPACE_DOMAIN:-localhost}${FRACTURING_SPACE_PUBLIC_PORT-:8080}/login}"
export FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS="${FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS:-${FRACTURING_SPACE_OAUTH_LOGIN_UI_URL}}"
export FRACTURING_SPACE_AI_PORT="${FRACTURING_SPACE_AI_PORT:-8087}"
export FRACTURING_SPACE_AI_ADDR="${FRACTURING_SPACE_AI_ADDR:-localhost:8087}"
export FRACTURING_SPACE_AI_DB_PATH="${FRACTURING_SPACE_AI_DB_PATH:-data/ai.db}"
export FRACTURING_SPACE_NOTIFICATIONS_PORT="${FRACTURING_SPACE_NOTIFICATIONS_PORT:-8088}"
export FRACTURING_SPACE_NOTIFICATIONS_ADDR="${FRACTURING_SPACE_NOTIFICATIONS_ADDR:-localhost:8088}"
export FRACTURING_SPACE_USERHUB_PORT="${FRACTURING_SPACE_USERHUB_PORT:-8092}"
export FRACTURING_SPACE_USERHUB_ADDR="${FRACTURING_SPACE_USERHUB_ADDR:-localhost:8092}"
export FRACTURING_SPACE_USERHUB_GAME_ADDR="${FRACTURING_SPACE_USERHUB_GAME_ADDR:-localhost:8082}"
export FRACTURING_SPACE_USERHUB_SOCIAL_ADDR="${FRACTURING_SPACE_USERHUB_SOCIAL_ADDR:-localhost:8090}"
export FRACTURING_SPACE_USERHUB_NOTIFICATIONS_ADDR="${FRACTURING_SPACE_USERHUB_NOTIFICATIONS_ADDR:-localhost:8088}"
export FRACTURING_SPACE_WORKER_PORT="${FRACTURING_SPACE_WORKER_PORT:-8089}"
export FRACTURING_SPACE_WORKER_AUTH_ADDR="${FRACTURING_SPACE_WORKER_AUTH_ADDR:-localhost:8083}"
export FRACTURING_SPACE_WORKER_GAME_ADDR="${FRACTURING_SPACE_WORKER_GAME_ADDR:-localhost:8082}"
export FRACTURING_SPACE_WORKER_SOCIAL_ADDR="${FRACTURING_SPACE_WORKER_SOCIAL_ADDR:-localhost:8090}"
export FRACTURING_SPACE_WORKER_NOTIFICATIONS_ADDR="${FRACTURING_SPACE_WORKER_NOTIFICATIONS_ADDR:-localhost:8088}"
export FRACTURING_SPACE_STATUS_PORT="${FRACTURING_SPACE_STATUS_PORT:-8093}"
export FRACTURING_SPACE_STATUS_ADDR="${FRACTURING_SPACE_STATUS_ADDR:-localhost:8093}"
export FRACTURING_SPACE_STATUS_DB_PATH="${FRACTURING_SPACE_STATUS_DB_PATH:-data/status.db}"

pids=()
cleanup_pids=()

start_service() {
  local name="$1"
  shift
  : > ".tmp/dev/${name}.log"
  env "$@" air -c ".devcontainer/air/${name}.toml" >> ".tmp/dev/${name}.log" 2>&1 &
  pids+=("$!")
  cleanup_pids+=("$!")
  printf 'started %s watcher (pid %s)\n' "$name" "$!"
}

start_background_command() {
  local name="$1"
  shift
  : > ".tmp/dev/${name}.log"
  "$@" >> ".tmp/dev/${name}.log" 2>&1 &
  pids+=("$!")
  cleanup_pids+=("$!")
  printf 'started %s process (pid %s)\n' "$name" "$!"
}

wait_for_service_log_marker() {
  local name="$1"
  local marker="$2"
  local max_attempts="${DEVCONTAINER_DEPENDENCY_READY_MAX_ATTEMPTS:-120}"
  local sleep_seconds="${DEVCONTAINER_DEPENDENCY_READY_SLEEP_SECONDS:-1}"
  local attempt=1
  local log_file=".tmp/dev/${name}.log"

  if [[ ! "$max_attempts" =~ ^[0-9]+$ ]] || (( max_attempts < 1 )); then
    echo "DEVCONTAINER_DEPENDENCY_READY_MAX_ATTEMPTS must be a positive integer" >&2
    return 1
  fi
  if [[ ! "$sleep_seconds" =~ ^[0-9]+$ ]] || (( sleep_seconds < 1 )); then
    echo "DEVCONTAINER_DEPENDENCY_READY_SLEEP_SECONDS must be a positive integer" >&2
    return 1
  fi

  while (( attempt <= max_attempts )); do
    if [[ -f "$log_file" ]] && tail -n 200 "$log_file" | grep -Fq "$marker"; then
      printf '%s readiness marker detected.\n' "$name"
      return 0
    fi

    if (( attempt == 1 || attempt == max_attempts || attempt % 10 == 0 )); then
      printf 'waiting for %s readiness marker (%d/%d)\n' "$name" "$attempt" "$max_attempts"
    fi

    attempt=$((attempt + 1))
    sleep "$sleep_seconds"
  done

  printf 'timed out waiting for %s readiness marker: %s\n' "$name" "$marker" >&2
  return 1
}

wait_for_http_ready() {
  local name="$1"
  local url="$2"
  local max_attempts="${DEVCONTAINER_DEPENDENCY_READY_MAX_ATTEMPTS:-120}"
  local sleep_seconds="${DEVCONTAINER_DEPENDENCY_READY_SLEEP_SECONDS:-1}"
  local attempt=1

  if [[ ! "$max_attempts" =~ ^[0-9]+$ ]] || (( max_attempts < 1 )); then
    echo "DEVCONTAINER_DEPENDENCY_READY_MAX_ATTEMPTS must be a positive integer" >&2
    return 1
  fi
  if [[ ! "$sleep_seconds" =~ ^[0-9]+$ ]] || (( sleep_seconds < 1 )); then
    echo "DEVCONTAINER_DEPENDENCY_READY_SLEEP_SECONDS must be a positive integer" >&2
    return 1
  fi

  while (( attempt <= max_attempts )); do
    if command -v curl >/dev/null 2>&1 && curl --silent --fail --max-time 2 "$url" >/dev/null 2>&1; then
      printf '%s readiness check passed.\n' "$name"
      return 0
    fi

    if (( attempt == 1 || attempt == max_attempts || attempt % 10 == 0 )); then
      printf 'waiting for %s at %s (%d/%d)\n' "$name" "$url" "$attempt" "$max_attempts"
    fi

    attempt=$((attempt + 1))
    sleep "$sleep_seconds"
  done

  printf 'timed out waiting for %s readiness at %s\n' "$name" "$url" >&2
  return 1
}

ensure_play_ui_node_modules() {
  local ui_root="internal/services/play/ui"
  local lockfile="$ui_root/package-lock.json"
  local modules_dir="$ui_root/node_modules"
  local stamp_file="$modules_dir/.package-lock.json"

  if [[ ! -f "$lockfile" ]]; then
    echo "missing $lockfile for play UI workspace" >&2
    exit 1
  fi

  if [[ ! -d "$modules_dir" || ! -f "$stamp_file" || "$lockfile" -nt "$stamp_file" ]]; then
    if ! command -v npm >/dev/null 2>&1; then
      echo "npm is required to install play UI workspace dependencies" >&2
      exit 1
    fi
    echo "installing play UI workspace dependencies"
    npm --prefix "$ui_root" ci
  fi
}

run_catalog_importer_async() {
  local log_file=".tmp/dev/catalog-importer.log"
  : > "$log_file"
  (
    {
      printf '[%s] starting async catalog importer\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
      go run ./cmd/catalog-importer \
        -dir internal/tools/importer/content/daggerheart/v1 \
        -db-path "${FRACTURING_SPACE_GAME_CONTENT_DB_PATH}" \
        -skip-if-ready
      printf '[%s] catalog importer finished successfully\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
    } >> "$log_file" 2>&1 || {
      printf '[%s] catalog importer failed\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" >> "$log_file"
      printf 'catalog importer failed; check %s\n' "$log_file" >&2
    }
  ) &
  cleanup_pids+=("$!")
  printf 'started catalog importer (pid %s)\n' "$!"
}

cleanup() {
  trap - EXIT INT TERM
  for pid in "${cleanup_pids[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
    fi
  done
  wait || true
  rm -f .tmp/dev/watch-services.pid
}

trap cleanup EXIT INT TERM

start_service status
start_service game FRACTURING_SPACE_GAME_ADDR=
start_service auth
start_service social
start_service discovery
start_service ai
start_service notifications
wait_for_service_log_marker "status" "status server listening at"
wait_for_service_log_marker "game" "game server listening"
run_catalog_importer_async
wait_for_service_log_marker "auth" "auth server listening at"
wait_for_service_log_marker "social" "social server listening at"
wait_for_service_log_marker "discovery" "discovery server listening at"
wait_for_service_log_marker "ai" "ai server listening at"
wait_for_service_log_marker "notifications" "notifications server listening at"
start_service userhub
wait_for_service_log_marker "userhub" "userhub server listening at"
start_service admin
start_service play
start_service worker
start_service web
ensure_play_ui_node_modules
start_background_command storybook npm --prefix internal/services/play/ui run storybook -- --host 0.0.0.0 --port 6006
wait_for_http_ready "storybook" "http://127.0.0.1:6006"

wait -n "${pids[@]}"
exit $?
