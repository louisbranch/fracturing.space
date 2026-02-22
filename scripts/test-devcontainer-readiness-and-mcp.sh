#!/usr/bin/env bash
set -euo pipefail

start_script=".devcontainer/scripts/start-devcontainer.sh"
watch_script=".devcontainer/scripts/watch-services.sh"

# Readiness polling should be configurable and less noisy by default.
if ! rg -n 'DEVCONTAINER_READY_MAX_ATTEMPTS' "$start_script" >/dev/null; then
  echo "expected configurable DEVCONTAINER_READY_MAX_ATTEMPTS in start-devcontainer.sh" >&2
  exit 1
fi
if ! rg -n 'DEVCONTAINER_READY_SLEEP_SECONDS' "$start_script" >/dev/null; then
  echo "expected configurable DEVCONTAINER_READY_SLEEP_SECONDS in start-devcontainer.sh" >&2
  exit 1
fi
if ! rg -n 'DEVCONTAINER_READY_LOG_EVERY' "$start_script" >/dev/null; then
  echo "expected configurable DEVCONTAINER_READY_LOG_EVERY in start-devcontainer.sh" >&2
  exit 1
fi
if ! rg -n 'attempt == 1 \|\| attempt % log_every == 0 \|\| attempt == max_attempts' "$start_script" >/dev/null; then
  echo "expected throttled readiness attempt logging in start-devcontainer.sh" >&2
  exit 1
fi

# MCP should not be started before game is ready.
if ! rg -n 'wait_for_service_log_marker "game" "game server listening at"' "$watch_script" >/dev/null; then
  echo "expected watch-services.sh to wait for game readiness before dependent services" >&2
  exit 1
fi
if ! rg -n 'start_service mcp' "$watch_script" >/dev/null; then
  echo "expected watch-services.sh to start mcp" >&2
  exit 1
fi
if ! rg -n 'ai-encryption\.key' "$watch_script" >/dev/null; then
  echo "expected watch-services.sh to persist a generated AI encryption key for dev use" >&2
  exit 1
fi
if rg -n 'MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY' "$watch_script" >/dev/null; then
  echo "expected watch-services.sh to avoid fixed, known AI encryption key defaults" >&2
  exit 1
fi
if ! rg -n 'if \[\[ ! "\$max_attempts" =~ \^\[0-9\]\+\$ \]\] \|\| \(\( max_attempts < 1 \)\); then' "$watch_script" >/dev/null; then
  echo "expected watch-services.sh to validate max_attempts override" >&2
  exit 1
fi
if ! rg -n 'if \[\[ ! "\$sleep_seconds" =~ \^\[0-9\]\+\$ \]\] \|\| \(\( sleep_seconds < 1 \)\); then' "$watch_script" >/dev/null; then
  echo "expected watch-services.sh to validate sleep_seconds override" >&2
  exit 1
fi

echo "PASS"
