#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"
repo_name="$(basename "$root_dir")"
export DEVCONTAINER_UID="${DEVCONTAINER_UID:-$(id -u)}"
export DEVCONTAINER_GID="${DEVCONTAINER_GID:-$(id -g)}"

set_devcontainer_user_env() {
  DEVCONTAINER_UID="${DEVCONTAINER_UID:-$(id -u)}"
  DEVCONTAINER_GID="${DEVCONTAINER_GID:-$(id -g)}"
  export DEVCONTAINER_UID DEVCONTAINER_GID
}

normalize_positive_int() {
  local raw="$1"
  local fallback="$2"
  if [[ ! "$raw" =~ ^[0-9]+$ ]] || (( raw < 1 )); then
    echo "$fallback"
    return
  fi
  echo "$raw"
}

wait_for_services_ready() {
  local max_attempts
  local sleep_seconds
  local log_every

  max_attempts="$(normalize_positive_int "${DEVCONTAINER_READY_MAX_ATTEMPTS:-180}" 180)"
  sleep_seconds="$(normalize_positive_int "${DEVCONTAINER_READY_SLEEP_SECONDS:-2}" 2)"
  log_every="$(normalize_positive_int "${DEVCONTAINER_READY_LOG_EVERY:-5}" 5)"

  local services=(
    "game"
    "auth"
    "connections"
    "ai"
    "notifications"
    "mcp"
    "admin"
    "chat"
    "worker"
    "web"
  )

  local markers=(
    "game server listening at"
    "auth server listening at"
    "connections server listening at"
    "ai server listening at"
    "notifications server listening at"
    "Starting MCP HTTP server"
    "admin listening on"
    "chat server listening on"
    "worker server listening at"
    "web server listening on"
  )

  local -a ready
  local -a remaining
  local log_file
  local i remaining_count ready_count
  local attempt=1
  local service_count="${#services[@]}"

  for i in "${!services[@]}"; do
    ready["$i"]=0
  done

  echo "Waiting for services to become ready (max_attempts=${max_attempts}, sleep_seconds=${sleep_seconds}, log_every=${log_every})..."

  while (( attempt <= max_attempts )); do
    ready_count=0
    remaining=()
    for i in "${!services[@]}"; do
      if [[ "${ready[$i]}" == "1" ]]; then
        ready_count=$((ready_count + 1))
        continue
      fi

      log_file=".tmp/dev/${services[$i]}.log"
      if [[ -f "$log_file" ]] && tail -n 200 "$log_file" | grep -Fq "${markers[$i]}"; then
        ready["$i"]=1
        ready_count=$((ready_count + 1))
        printf '%s service is ready.\n' "${services[$i]}"
      else
        remaining+=("${services[$i]}")
      fi
    done

    if (( ready_count == service_count )); then
      echo "All dev services are ready."
      return 0
    fi

    if (( attempt == 1 || attempt % log_every == 0 || attempt == max_attempts )); then
      remaining_count="${#remaining[@]}"
      printf '  attempt %d/%d: waiting for %d/%d services (%s)\n' \
        "$attempt" "$max_attempts" "$remaining_count" "$service_count" \
        "$(printf '%s, ' "${remaining[@]}" | sed 's/, $//')"
    fi

    attempt=$((attempt + 1))
    sleep "$sleep_seconds"
  done

  echo "Timed out waiting for services to be ready; check .tmp/dev/*.log for details." >&2
  return 1
}

run_post_start_in_container() {
  docker compose --env-file .env -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer bash -lc "set -euo pipefail; if [ -d /workspace/${repo_name} ]; then cd /workspace/${repo_name}; else cd /workspace; fi; if [ ! -f .devcontainer/scripts/post-start.sh ]; then echo '.devcontainer/scripts/post-start.sh not found in container workspace' >&2; exit 1; fi; bash .devcontainer/scripts/post-start.sh"
}

wait_for_devcontainer_ready() {
  for _ in $(seq 1 20); do
    if docker compose --env-file .env -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer true >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "devcontainer did not become ready for exec commands" >&2
  return 1
}

ensure_go_toolchain() {
  if docker compose --env-file .env -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer bash -lc "(command -v go >/dev/null 2>&1 || [ -x /usr/local/go/bin/go ]) && (command -v air >/dev/null 2>&1 || [ -x /go/bin/air ] || [ -x /root/go/bin/air ])"; then
    return 0
  fi

  echo "devcontainer image missing required dev tooling (go/air); rebuilding devcontainer service" >&2
  docker compose --env-file .env -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml up -d --build devcontainer
  wait_for_devcontainer_ready

  if docker compose --env-file .env -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer bash -lc "(command -v go >/dev/null 2>&1 || [ -x /usr/local/go/bin/go ]) && (command -v air >/dev/null 2>&1 || [ -x /go/bin/air ] || [ -x /root/go/bin/air ])"; then
    return 0
  fi

  echo "rebuilt devcontainer is still missing required tooling (go/air)" >&2
  return 1
}

if [[ -f "/.dockerenv" ]]; then
  bash .devcontainer/scripts/post-start.sh
  exit 0
fi

set_devcontainer_user_env

BOOTSTRAP_SKIP_UP=1 ./scripts/bootstrap.sh

docker compose --env-file .env -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml up -d devcontainer
wait_for_devcontainer_ready
ensure_go_toolchain
run_post_start_in_container
wait_for_services_ready
