#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

env_file="${ENV_FILE:-$root_dir/.env.production}"
env_example="${ENV_EXAMPLE:-$root_dir/.env.production.example}"

if [[ ! -f "$env_file" ]]; then
  cp "$env_example" "$env_file"
  printf 'Created %s from %s\n' "$env_file" "$env_example"
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
  local line
  local replaced=0

  tmp="$(mktemp)"
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
  trap - EXIT
}

generate_join_grant_keys() {
  local output
  local line
  local key
  local value
  local found=0

  output="$(go run ./cmd/join-grant-key)"
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

  output="$(go run ./cmd/hmac-key)"
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
  printf 'Generated game event HMAC key\n'
}

generate_random_base64() {
  dd if=/dev/urandom bs=32 count=1 2>/dev/null | base64 | tr -d '\r\n'
}

ensure_base64_secret() {
  local key="$1"
  local current
  local next

  current="$(get_env_value "$key")"
  if [[ -n "$current" ]]; then
    return 0
  fi

  next="$(generate_random_base64)"
  if [[ -z "$next" ]]; then
    printf 'Error: failed to generate %s.\n' "$key" >&2
    exit 1
  fi

  set_env_value "$key" "$next"
  printf 'Generated %s\n' "$key"
}

ensure_plain_secret() {
  local key="$1"
  local current
  local next

  current="$(get_env_value "$key")"
  if [[ -n "$current" ]]; then
    return 0
  fi

  next="$(generate_random_base64)"
  if [[ -z "$next" ]]; then
    printf 'Error: failed to generate %s.\n' "$key" >&2
    exit 1
  fi

  set_env_value "$key" "$next"
  printf 'Generated %s\n' "$key"
}

hmac_key="$(get_env_value FRACTURING_SPACE_GAME_EVENT_HMAC_KEY)"
if [[ -z "$hmac_key" ]]; then
  generate_hmac_key
fi

join_public="$(get_env_value FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY)"
join_private="$(get_env_value FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY)"
if [[ -z "$join_public" || -z "$join_private" ]]; then
  generate_join_grant_keys
fi

ensure_base64_secret "FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY"
ensure_base64_secret "FRACTURING_SPACE_AI_ENCRYPTION_KEY"
ensure_base64_secret "FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY"
ensure_plain_secret "FRACTURING_SPACE_OAUTH_RESOURCE_SECRET"

printf '\nRemaining required values to fill manually in %s:\n' "$env_file"
required_keys=(
  "FRACTURING_SPACE_IMAGE_TAG"
  "FRACTURING_SPACE_DOMAIN"
  "FRACTURING_SPACE_DAGGERHEART_REFERENCE_IMAGE"
  "FRACTURING_SPACE_GAME_EVENT_HMAC_KEY"
  "FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY"
  "FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY"
  "FRACTURING_SPACE_PLAY_LAUNCH_GRANT_HMAC_KEY"
  "FRACTURING_SPACE_OAUTH_RESOURCE_SECRET"
  "FRACTURING_SPACE_AI_ENCRYPTION_KEY"
  "FRACTURING_SPACE_AI_SESSION_GRANT_HMAC_KEY"
  "FRACTURING_SPACE_JAEGER_BASIC_AUTH"
  "FRACTURING_SPACE_OPENVIKING_OPENAI_API_KEY"
)

for key in "${required_keys[@]}"; do
  value="$(get_env_value "$key")"
  if [[ -z "$value" || "$value" == REPLACE_WITH_* ]]; then
    printf '  %s\n' "$key"
  fi
done
