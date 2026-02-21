#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! grep -Fq 'export DEVCONTAINER_UID="${DEVCONTAINER_UID:-$(id -u)}"' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to default DEVCONTAINER_UID to host uid" >&2
  exit 1
fi

if ! grep -Fq 'export DEVCONTAINER_GID="${DEVCONTAINER_GID:-$(id -g)}"' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to default DEVCONTAINER_GID to host gid" >&2
  exit 1
fi

if ! grep -Fq 'export DEVCONTAINER_UID="${DEVCONTAINER_UID:-$(id -u)}"' .devcontainer/scripts/stop-devcontainer.sh; then
  echo "expected stop-devcontainer.sh to default DEVCONTAINER_UID to host uid" >&2
  exit 1
fi

if ! grep -Fq 'export DEVCONTAINER_GID="${DEVCONTAINER_GID:-$(id -g)}"' .devcontainer/scripts/stop-devcontainer.sh; then
  echo "expected stop-devcontainer.sh to default DEVCONTAINER_GID to host gid" >&2
  exit 1
fi

if ! grep -Fq 'HOME: /home/vscode' .devcontainer/docker-compose.devcontainer.yml; then
  echo "expected devcontainer HOME to be /home/vscode" >&2
  exit 1
fi

if grep -Fq 'HOME: /workspace' .devcontainer/docker-compose.devcontainer.yml; then
  echo "expected devcontainer HOME to avoid /workspace" >&2
  exit 1
fi

if ! grep -Fq 'devcontainer-home:/home/vscode' .devcontainer/docker-compose.devcontainer.yml; then
  echo "expected devcontainer compose to mount a dedicated home volume" >&2
  exit 1
fi

echo "PASS"
