#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

log() {
  printf '[worktree-pre-delete] %s\n' "$1"
}

run_best_effort() {
  local label="$1"
  shift

  if "$@" >/dev/null 2>&1; then
    log "$label"
    return 0
  fi

  return 1
}

remove_path() {
  local path="$1"

  if [[ ! -e "$path" && ! -L "$path" ]]; then
    return 0
  fi

  rm -rf -- "$path"
  log "removed $path"
}

cleanup_data_runtime() {
  local path

  if [[ ! -d data ]]; then
    return 0
  fi

  shopt -s nullglob dotglob
  for path in data/*; do
    if [[ "$(basename "$path")" == "instructions" ]]; then
      continue
    fi

    rm -rf -- "$path"
    log "removed $path"
  done
  shopt -u nullglob dotglob
}

# Stop repo-managed watcher processes first so their traps can clean child
# processes before the worktree disappears underneath them.
run_best_effort "stopped watch-services" bash .devcontainer/scripts/stop-watch-services.sh || true

# Then stop the devcontainer service if one is active for this worktree.
run_best_effort "stopped devcontainer" bash .devcontainer/scripts/stop-devcontainer.sh || true

cleanup_data_runtime
remove_path .tmp
remove_path .config
remove_path .playwright-cli
remove_path artifacts
remove_path seed
remove_path .env
remove_path .env.local
