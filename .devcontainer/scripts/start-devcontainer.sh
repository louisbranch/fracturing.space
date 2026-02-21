#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root_dir"
repo_name="$(basename "$root_dir")"

run_post_start_in_container() {
  docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer bash -lc "set -euo pipefail; if [ -d /workspace/${repo_name} ]; then cd /workspace/${repo_name}; else cd /workspace; fi; if [ ! -f .devcontainer/scripts/post-start.sh ]; then echo '.devcontainer/scripts/post-start.sh not found in container workspace' >&2; exit 1; fi; bash .devcontainer/scripts/post-start.sh"
}

wait_for_devcontainer_ready() {
  for _ in $(seq 1 20); do
    if docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer true >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  echo "devcontainer did not become ready for exec commands" >&2
  return 1
}

ensure_go_toolchain() {
  if docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer bash -lc "(command -v go >/dev/null 2>&1 || [ -x /usr/local/go/bin/go ]) && (command -v air >/dev/null 2>&1 || [ -x /go/bin/air ] || [ -x /root/go/bin/air ])"; then
    return 0
  fi

  echo "devcontainer image missing required dev tooling (go/air); rebuilding devcontainer service" >&2
  docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml up -d --build devcontainer
  wait_for_devcontainer_ready

  if docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer bash -lc "(command -v go >/dev/null 2>&1 || [ -x /usr/local/go/bin/go ]) && (command -v air >/dev/null 2>&1 || [ -x /go/bin/air ] || [ -x /root/go/bin/air ])"; then
    return 0
  fi

  echo "rebuilt devcontainer is still missing required tooling (go/air)" >&2
  return 1
}

if [[ -f "/.dockerenv" ]]; then
  bash .devcontainer/scripts/post-start.sh
  exit 0
fi

docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml up -d devcontainer
wait_for_devcontainer_ready
ensure_go_toolchain
run_post_start_in_container
