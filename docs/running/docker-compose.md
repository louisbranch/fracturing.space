---
title: "Docker Compose"
parent: "Running"
nav_order: 3
---

# Docker Compose (Local)

## Start the stack

For the one-line command, see [quickstart](quickstart.md).

## Local routes

- Web login: `http://localhost:8080`
- Auth: `http://auth.localhost:8080`
- Admin: `http://admin.localhost:8080`
- MCP health: `http://mcp.localhost:8080/mcp/health`

## Configuration

Compose reads `.env` if present.

Key settings:

- `FRACTURING_SPACE_DOMAIN` (default `localhost`)
- `FRACTURING_SPACE_PUBLIC_PORT` (default `:8080`)
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY` (defaults to `dev-secret`)
- `FRACTURING_SPACE_AI_ENCRYPTION_KEY` (required; `make bootstrap` auto-generates when missing)
- Join-grant keys (dev-only defaults in `docker-compose.yml` and `.env.example`)
- WebAuthn passkey config (Compose provides defaults matching the local domain and port)

See [configuration](configuration.md) for the full list.

For production, see [production](production.md).

## Tools

Compose exposes CLI tools under the `tools` profile:

```sh
docker compose --profile tools run --rm hmac-key
docker compose --profile tools run --rm join-grant-key
docker compose --profile tools run --rm seed
docker compose --profile tools run --rm seed -- -generate -preset=variety -v
docker compose --profile tools run --rm scenario -- -scenario internal/test/game/scenarios/basic_flow.lua
docker compose --profile tools run --rm maintenance -- -campaign-id <id> -validate
docker compose --profile tools run --rm catalog-importer
```

## Volumes

Compose uses named volumes for SQLite data stores. To remove them:

```sh
docker compose down -v
```
