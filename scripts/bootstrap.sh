#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

env_file="${ENV_FILE:-$root_dir/.env}"
env_example="${ENV_EXAMPLE:-$root_dir/.env.example}"

if [[ ! -f "$env_file" ]]; then
  cp "$env_example" "$env_file"
  printf 'Created %s from %s\n' "$env_file" "$env_example"
fi

if [[ -n "${COMPOSE_CMD:-}" ]]; then
  read -r -a compose_cmd <<< "${COMPOSE_CMD}"
else
  compose_cmd=(
    docker compose
    -f docker-compose.yml
    -f topology/generated/docker-compose.discovery.generated.yml
  )
fi

get_env_value() {
  local key="$1"
  local value=""
  local line

  while IFS= read -r line; do
    case "$line" in
      "${key}="*)
        value="${line#*=}"
        ;;
    esac
  done < "$env_file"

  printf '%s' "$value"
}

set_env_value() {
  local key="$1"
  local value="$2"
  local tmp
  local exit_trap
  local replaced=0
  local line

  tmp="$(mktemp)"
  exit_trap="$(trap -p EXIT)"
  trap 'rm -f "$tmp"' EXIT

  while IFS= read -r line; do
    if [[ "$line" == "${key}="* ]]; then
      printf '%s=%s\n' "$key" "$value" >> "$tmp"
      replaced=1
      continue
    fi
    printf '%s\n' "$line" >> "$tmp"
  done < "$env_file"

  if [[ "$replaced" -eq 0 ]]; then
    printf '%s=%s\n' "$key" "$value" >> "$tmp"
  fi

  mv "$tmp" "$env_file"

  if [[ -n "$exit_trap" ]]; then
    eval "$exit_trap"
  else
    trap - EXIT
  fi
}

generate_join_grant_keys() {
  local output
  local line
  local key
  local value
  local found=0

  output="$(${compose_cmd[@]} --profile tools run --rm join-grant-key)"
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    case "$line" in
      FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY=*|FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY=*)
        key="${line%%=*}"
        value="${line#*=}"
        set_env_value "$key" "$value"
        found=1
        ;;
    esac
  done <<< "$output"
  if [[ "$found" -eq 0 ]]; then
    printf 'Error: join-grant-key output did not include expected keys.\n' >&2
    exit 1
  fi
  printf 'Generated join-grant keys\n'
}

generate_hmac_key() {
  local output
  local line
  local key
  local value
  local found=0

  output="$(${compose_cmd[@]} --profile tools run --rm hmac-key)"
  while IFS= read -r line; do
    [[ -z "$line" ]] && continue
    case "$line" in
      FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=*)
        key="${line%%=*}"
        value="${line#*=}"
        set_env_value "$key" "$value"
        found=1
        ;;
    esac
  done <<< "$output"
  if [[ "$found" -eq 0 ]]; then
    printf 'Error: hmac-key output did not include the expected key.\n' >&2
    exit 1
  fi
  printf 'Generated HMAC key\n'
}

generate_ai_encryption_key() {
  local value

  value="$(dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\r\n')"
  if [[ -z "$value" ]]; then
    printf 'Error: failed to generate FRACTURING_SPACE_AI_ENCRYPTION_KEY.\n' >&2
    exit 1
  fi

  set_env_value "FRACTURING_SPACE_AI_ENCRYPTION_KEY" "$value"
  printf 'Generated AI encryption key\n'
}

ai_encryption_key="$(get_env_value FRACTURING_SPACE_AI_ENCRYPTION_KEY)"
if [[ -z "$ai_encryption_key" || "$ai_encryption_key" == "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY" ]]; then
  generate_ai_encryption_key
fi

join_public="$(get_env_value FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY)"
join_private="$(get_env_value FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY)"
if [[ -z "$join_public" || -z "$join_private" ]]; then
  generate_join_grant_keys
fi

hmac_keys="$(get_env_value FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS)"
hmac_key="$(get_env_value FRACTURING_SPACE_GAME_EVENT_HMAC_KEY)"
if [[ -z "$hmac_keys" ]]; then
  if [[ -z "$hmac_key" || "$hmac_key" == "dev-secret" ]]; then
    generate_hmac_key
  fi
fi

if [[ "${BOOTSTRAP_SKIP_UP:-}" == "1" ]]; then
  printf 'Skipping docker compose up (BOOTSTRAP_SKIP_UP=1)\n'
  exit 0
fi

${compose_cmd[@]} up -d
