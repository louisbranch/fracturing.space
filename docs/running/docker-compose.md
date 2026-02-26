---
title: "Docker Compose"
parent: "Running"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Docker Compose (Local)

## Start the stack

For the one-line command, see [quickstart](quickstart.md).

Compose commands should include both the base file and generated topology discovery file:

```sh
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml up -d
```

Compose + Caddy topology is catalog-driven:

- Source of truth: `topology/services.json`
- Generated Caddy routes: `Caddyfile.routes.generated`
- Generated Compose discovery artifact: `topology/generated/docker-compose.discovery.generated.yml`

After topology edits, regenerate and validate:

```sh
make topology-generate
make topology-check
```

## Local routes

- Web login: `http://localhost:8080`
- Auth: `http://auth.localhost:8080`
- Admin: `http://admin.localhost:8080`
- MCP health: `http://mcp.localhost:8080/mcp/health`
- Notifications gRPC (internal): `notifications:8088`
- Worker gRPC health (internal): `worker:8089`

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
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml --profile tools run --rm hmac-key
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml --profile tools run --rm join-grant-key
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml --profile tools run --rm seed
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml --profile tools run --rm scenario -- -scenario internal/test/game/scenarios/basic_flow.lua
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml --profile tools run --rm maintenance -- -campaign-id <id> -validate
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml --profile tools run --rm catalog-importer
```

The `seed` tool in this repository is intentionally scoped to the local-dev manifest for local workflows. For any production-like environment, use a separate migration/administrative flow instead of `seed`.

## Volumes

Compose uses named volumes for SQLite data stores. To remove them:

```sh
docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml down -v
```
