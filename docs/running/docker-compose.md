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

Compose reads `.env` if present. Defaults are safe for local development.

Key settings:

- `FRACTURING_SPACE_DOMAIN` (default `localhost`)
- `FRACTURING_SPACE_PUBLIC_PORT` (default `:8080`)
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY` (defaults to `dev-secret`)
- Join-grant keys (dev-only defaults in `docker-compose.yml` and `.env.example`)

See [configuration](configuration.md) for the full list.

For production, see [production](production.md).

## Volumes

Compose uses named volumes for SQLite data stores. To remove them:

```sh
docker compose down -v
```
