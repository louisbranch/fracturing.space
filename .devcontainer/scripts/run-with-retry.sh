#!/usr/bin/env bash
# run-with-retry.sh — wraps a binary with retry-on-failure for dev use.
#
# Air calls this instead of the binary directly. On non-zero exit the script
# retries with exponential backoff so that a transient dependency-dial timeout
# (e.g. game starting before auth is ready) doesn't leave a service dead until
# the next file change.
#
# Usage (in air toml):
#   bin = "bash .devcontainer/scripts/run-with-retry.sh .tmp/dev/bin/game"
#
# Environment tunables:
#   DEV_RETRY_MAX       — max retry attempts (default 5)
#   DEV_RETRY_DELAY     — initial delay in seconds (default 2)
#   DEV_RETRY_MAX_DELAY — cap on backoff delay in seconds (default 15)

set -uo pipefail

binary="$1"
shift

max_retries="${DEV_RETRY_MAX:-5}"
delay="${DEV_RETRY_DELAY:-2}"
max_delay="${DEV_RETRY_MAX_DELAY:-15}"
child_pid=""

# Forward signals to the child so air can cleanly stop the service.
forward_signal() {
  if [[ -n "$child_pid" ]] && kill -0 "$child_pid" 2>/dev/null; then
    kill -"$1" "$child_pid" 2>/dev/null || true
  fi
}
trap 'forward_signal TERM; exit 143' TERM
trap 'forward_signal INT;  exit 130' INT

attempt=0
while true; do
  "$binary" "$@" &
  child_pid=$!
  wait "$child_pid"
  rc=$?
  child_pid=""

  # Clean exit or killed by signal forwarded above.
  if (( rc == 0 || rc >= 128 )); then
    exit "$rc"
  fi

  attempt=$((attempt + 1))

  if (( attempt >= max_retries )); then
    printf '[run-with-retry] %s exited %d after %d attempts, giving up\n' \
      "$binary" "$rc" "$attempt" >&2
    exit "$rc"
  fi

  printf '[run-with-retry] %s exited %d, retrying in %ds (attempt %d/%d)\n' \
    "$binary" "$rc" "$delay" "$attempt" "$max_retries" >&2
  sleep "$delay"

  # Exponential backoff capped at max_delay.
  delay=$(( delay * 2 ))
  if (( delay > max_delay )); then
    delay=$max_delay
  fi
done
