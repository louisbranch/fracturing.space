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

if [[ -f "/.dockerenv" ]]; then
  bash .devcontainer/scripts/stop-watch-services.sh
  exit 0
fi

set_devcontainer_user_env

docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml exec -T devcontainer bash -lc "set -euo pipefail; if [ -d /workspace/${repo_name} ]; then cd /workspace/${repo_name}; else cd /workspace; fi; if [ ! -f .devcontainer/scripts/stop-watch-services.sh ]; then exit 0; fi; bash .devcontainer/scripts/stop-watch-services.sh" >/dev/null 2>&1 || true
docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml down
