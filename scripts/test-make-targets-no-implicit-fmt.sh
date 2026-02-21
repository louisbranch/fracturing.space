#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

proto_plan="$(make -n proto)"
templ_plan="$(make -n templ-generate)"
setup_plan="$(make -n setup-hooks)"
up_plan="$(make -n up)"
down_plan="$(make -n down)"

if grep -Fq 'goimports -w' <<<"$proto_plan"; then
  echo "expected proto target to avoid implicit formatting commands" >&2
  exit 1
fi

if grep -Fq 'goimports -w' <<<"$templ_plan"; then
  echo "expected templ-generate target to avoid implicit formatting commands" >&2
  exit 1
fi

if ! grep -Fq 'git config --local --get core.hooksPath' <<<"$setup_plan"; then
  echo "expected setup-hooks to check existing hooks path before writing" >&2
  exit 1
fi

if ! grep -Fq 'already configured' <<<"$setup_plan"; then
  echo "expected setup-hooks to no-op when already configured" >&2
  exit 1
fi

if ! grep -Fq 'chmod +x .githooks/pre-commit' <<<"$setup_plan"; then
  echo "expected setup-hooks to ensure executable pre-commit hook" >&2
  exit 1
fi

if make -n run >/tmp/make-run.out 2>/tmp/make-run.err; then
  echo "expected make run target to be removed" >&2
  exit 1
fi

if ! grep -Fq 'bash .devcontainer/scripts/start-devcontainer.sh' <<<"$up_plan"; then
  echo "expected up target to call start-devcontainer.sh" >&2
  exit 1
fi

if ! grep -Fq 'bash .devcontainer/scripts/stop-devcontainer.sh' <<<"$down_plan"; then
  echo "expected down target to call stop-devcontainer.sh" >&2
  exit 1
fi

if ! grep -Fq 'docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml up -d devcontainer' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to bring up the devcontainer service with devcontainer compose file first" >&2
  exit 1
fi

if ! grep -Fq '.devcontainer/scripts/post-start.sh' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to include watcher startup" >&2
  exit 1
fi

if ! grep -Fq '/workspace/${repo_name}' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to resolve repo path when /workspace is a parent mount" >&2
  exit 1
fi

if ! grep -Fq 'command -v go >/dev/null 2>&1' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to validate go is available in the devcontainer" >&2
  exit 1
fi

if ! grep -Fq '/usr/local/go/bin/go' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to treat /usr/local/go/bin/go as a valid toolchain location" >&2
  exit 1
fi

if ! grep -Fq 'command -v air >/dev/null 2>&1' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to validate air is available in the devcontainer" >&2
  exit 1
fi

if ! grep -Fq '/go/bin/air' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to treat /go/bin/air as a valid air location" >&2
  exit 1
fi

if ! grep -Fq 'up -d --build devcontainer' .devcontainer/scripts/start-devcontainer.sh; then
  echo "expected start-devcontainer.sh to rebuild devcontainer when toolchain is missing" >&2
  exit 1
fi

if ! grep -Fq '/usr/local/go/bin' .devcontainer/scripts/watch-services.sh; then
  echo "expected watch-services.sh to include /usr/local/go/bin in PATH before calling go" >&2
  exit 1
fi

if ! grep -Fq 'go install github.com/air-verse/air@v1.62.0' Dockerfile; then
  echo "expected Dockerfile to preinstall air in the devcontainer image" >&2
  exit 1
fi

if grep -Fq 'go install github.com/air-verse/air@v1.62.0' .devcontainer/scripts/watch-services.sh; then
  echo "expected watch-services.sh to avoid runtime air installation" >&2
  exit 1
fi

if ! grep -Fq '.devcontainer/scripts/stop-watch-services.sh' .devcontainer/scripts/stop-devcontainer.sh; then
  echo "expected stop-devcontainer.sh to include watcher shutdown" >&2
  exit 1
fi

if ! grep -Fq '/workspace/${repo_name}' .devcontainer/scripts/stop-devcontainer.sh; then
  echo "expected stop-devcontainer.sh to resolve repo path when /workspace is a parent mount" >&2
  exit 1
fi

if ! grep -Fq 'docker compose -f .devcontainer/docker-compose.devcontainer.yml -f docker-compose.yml down' .devcontainer/scripts/stop-devcontainer.sh; then
  echo "expected stop-devcontainer.sh to bring down devcontainer compose services with devcontainer compose file first" >&2
  exit 1
fi

if ! grep -Fq 'default' .devcontainer/docker-compose.devcontainer.yml; then
  echo "expected devcontainer service to include a non-internal network so published localhost ports work" >&2
  exit 1
fi

echo "PASS"
