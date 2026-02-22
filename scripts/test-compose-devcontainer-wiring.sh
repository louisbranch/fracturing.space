#!/usr/bin/env bash
set -euo pipefail

# Compose must include the AI service image target.
if ! rg -n "^  ai:" docker-compose.yml >/dev/null; then
  echo "expected docker-compose.yml to define an ai service" >&2
  exit 1
fi
if ! rg -n "target: ai" docker-compose.yml >/dev/null; then
  echo "expected ai service to build from Dockerfile target ai" >&2
  exit 1
fi

# Dockerfile must support building/running AI images for compose.
if ! rg -n "^FROM base AS build-ai$" Dockerfile >/dev/null; then
  echo "expected Dockerfile build-ai stage" >&2
  exit 1
fi
if ! rg -n "^FROM gcr.io/distroless/static-debian12:nonroot AS ai$" Dockerfile >/dev/null; then
  echo "expected Dockerfile ai runtime stage" >&2
  exit 1
fi

# Requested primary local port ordering.
if ! rg -n "FRACTURING_SPACE_WEB_HTTP_ADDR: 0.0.0.0:8080" docker-compose.yml >/dev/null; then
  echo "expected web to listen on :8080 in compose" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_ADMIN_ADDR: 0.0.0.0:8081" docker-compose.yml >/dev/null; then
  echo "expected admin to listen on :8081 in compose" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_GAME_PORT: 8082" docker-compose.yml >/dev/null; then
  echo "expected game to listen on :8082 in compose" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_GAME_ADDR: game:8082" docker-compose.yml >/dev/null; then
  echo "expected compose dependencies to target game:8082" >&2
  exit 1
fi

# Remaining ports should continue in sequence for local/devcontainer use.
if ! rg -n "FRACTURING_SPACE_MCP_HTTP_ADDR: 0.0.0.0:8085" docker-compose.yml >/dev/null; then
  echo "expected mcp to listen on :8085 in compose" >&2
  exit 1
fi
if ! grep -Fq "FRACTURING_SPACE_CHAT_HTTP_ADDR: \${FRACTURING_SPACE_CHAT_HTTP_ADDR:-0.0.0.0:8086}" docker-compose.yml; then
  echo "expected chat default http addr to be :8086 in compose" >&2
  exit 1
fi
if ! grep -Fq "FRACTURING_SPACE_AI_PORT: \${FRACTURING_SPACE_AI_PORT:-8087}" docker-compose.yml; then
  echo "expected ai default grpc port to be :8087 in compose" >&2
  exit 1
fi
if ! grep -Fq "FRACTURING_SPACE_AI_ENCRYPTION_KEY: \${FRACTURING_SPACE_AI_ENCRYPTION_KEY?FRACTURING_SPACE_AI_ENCRYPTION_KEY must be set}" docker-compose.yml; then
  echo "expected compose to require explicit FRACTURING_SPACE_AI_ENCRYPTION_KEY" >&2
  exit 1
fi

# Devcontainer port forwarding must include ordered 8080..8087 range.
for port in 8080 8081 8082 8083 8084 8085 8086 8087; do
  if ! rg -n "\\b${port}\\b" .devcontainer/devcontainer.json >/dev/null; then
    echo "expected devcontainer forwardPorts to include ${port}" >&2
    exit 1
  fi
  if ! rg -n "\"${port}:${port}\"" .devcontainer/docker-compose.devcontainer.yml >/dev/null; then
    echo "expected devcontainer compose ports to include ${port}:${port}" >&2
    exit 1
  fi
done

# Watcher runtime must include chat + ai and matching local addresses.
if ! rg -n "start_service chat" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh to start chat" >&2
  exit 1
fi
if ! rg -n "start_service ai" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh to start ai" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_WEB_HTTP_ADDR.*8080" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh web address to default to :8080" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_ADMIN_ADDR.*8081" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh admin address to default to :8081" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_GAME_PORT.*8082" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh game port to default to :8082" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_MCP_HTTP_ADDR.*8085" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh mcp address to default to :8085" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_CHAT_HTTP_ADDR.*8086" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh chat address to default to :8086" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_AI_PORT.*8087" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh ai port to default to :8087" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_OAUTH_LOGIN_UI_URL.*FRACTURING_SPACE_PUBLIC_PORT-:8080.*/login" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh oauth login UI URL to include public port fallback (:8080) and /login path" >&2
  exit 1
fi
if ! rg -n "FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS.*FRACTURING_SPACE_OAUTH_LOGIN_UI_URL" .devcontainer/scripts/watch-services.sh >/dev/null; then
  echo "expected watch-services.sh oauth login redirects to default to oauth login UI URL" >&2
  exit 1
fi

# Startup readiness should wait for chat + ai logs too.
if ! rg -n '"chat"' .devcontainer/scripts/start-devcontainer.sh >/dev/null; then
  echo "expected start-devcontainer.sh readiness list to include chat" >&2
  exit 1
fi
if ! rg -n '"ai"' .devcontainer/scripts/start-devcontainer.sh >/dev/null; then
  echo "expected start-devcontainer.sh readiness list to include ai" >&2
  exit 1
fi
if ! rg -n "chat server listening on" .devcontainer/scripts/start-devcontainer.sh >/dev/null; then
  echo "expected start-devcontainer.sh to track chat readiness marker" >&2
  exit 1
fi
if ! rg -n "ai server listening at" .devcontainer/scripts/start-devcontainer.sh >/dev/null; then
  echo "expected start-devcontainer.sh to track ai readiness marker" >&2
  exit 1
fi

echo "PASS"
