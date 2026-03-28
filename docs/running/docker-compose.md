---
title: "Docker Compose"
parent: "Running"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-27"
---

# Docker Compose (Local)

## Start the stack

For the one-line command, see [quickstart](quickstart.md).

Compose commands should include both the base file and generated topology discovery file:

```sh
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml up -d
```

Compose + Caddy topology is catalog-driven:

- Source of truth: `topology/services.json`
- Generated Caddy routes: `Caddyfile.routes.generated`
- Generated Compose discovery artifact: `topology/generated/docker-compose.serviceaddr.generated.yml`

After topology edits, regenerate and validate:

```sh
make topology-generate
make topology-check
```

## Local routes

- Web login: `http://localhost:8080`
- Auth: `http://auth.localhost:8080`
- Admin: `http://admin.localhost:8080`
- Notifications gRPC (internal): `notifications:8088`
- Worker gRPC health (internal): `worker:8089`
- OpenViking host port when enabled: `http://127.0.0.1:1933`

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

Production does not reuse the local Compose topology file. Remote deployment
uses [`docker-compose.production.yml`](../../docker-compose.production.yml),
which is image-only and relies on named volumes plus fixed container paths
instead of local bind mounts.

## Optional OpenViking Sidecar

OpenViking is available as an opt-in Compose profile for local evaluation. It
is not part of the default stack. The local evaluation path is pinned to
`ghcr.io/volcengine/openviking:v0.2.10`.

Prepare the local host paths first:

```sh
mkdir -p ~/.openviking/data
cp docker/openviking/ov.conf.example ~/.openviking/ov.conf
```

Then edit `~/.openviking/ov.conf` and replace the placeholder OpenAI API keys.
The tracked example now reflects the docs-aligned OpenAI profile used for the
next evaluation phase:

- embedding: `text-embedding-3-large`
- VLM: `gpt-4o`

Start only the sidecar:

```sh
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml --profile openviking up -d openviking
```

The setup exposes two useful URLs:

- host-run tools and live tests: `http://127.0.0.1:1933`
- the Compose `ai` container: `http://openviking:1934`

If you want the Compose `ai` container to use OpenViking, set this in `.env`:

```sh
FRACTURING_SPACE_AI_OPENVIKING_BASE_URL=http://openviking:1934
FRACTURING_SPACE_AI_OPENVIKING_MODE=legacy
FRACTURING_SPACE_AI_OPENVIKING_SESSION_SYNC_ENABLED=true
FRACTURING_SPACE_AI_OPENVIKING_MIRROR_ROOT=/openviking-data/fracturing-space
FRACTURING_SPACE_AI_OPENVIKING_VISIBLE_MIRROR_ROOT=/app/data/fracturing-space
```

Use `FRACTURING_SPACE_AI_OPENVIKING_MODE=docs_aligned_supplement` when running
the docs-aligned evaluation mode that suppresses raw `story.md` but keeps raw
`memory.md` in the prompt.

For host-run live tests, keep the sidecar URL on `http://127.0.0.1:1933` and
set the mirror roots like this:

```sh
FRACTURING_SPACE_AI_OPENVIKING_MIRROR_ROOT=$HOME/.openviking/data/fracturing-space
FRACTURING_SPACE_AI_OPENVIKING_VISIBLE_MIRROR_ROOT=/app/data/fracturing-space
```

The live AI capture lane now defaults to augmentation-only evaluation when
OpenViking is enabled: it disables session sync unless you explicitly set
`FRACTURING_SPACE_AI_OPENVIKING_SESSION_SYNC_ENABLED=true`, and it raises the
resource-ingest timeout to `20s` unless you already set
`FRACTURING_SPACE_AI_OPENVIKING_RESOURCE_SYNC_TIMEOUT`.

This profile uses the repo's Python TCP forwarder so the service remains
reachable even though upstream OpenViking defaults to listening on
`127.0.0.1` inside the container. The Compose setup also
places OpenViking on the non-internal `edge` network so it can reach OpenAI for
embedding and VLM calls while still sharing the internal network with the AI
service.

This host-path setup is local-only. The production deployment path replaces it
with a repo-owned `openviking-sidecar` image plus named volumes so the remote
server does not need `~/.openviking` or any checkout-relative mount layout.

## Tools

Compose exposes CLI tools under the `tools` profile:

```sh
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml --profile tools run --rm hmac-key
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml --profile tools run --rm join-grant-key
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml --profile tools run --rm seed
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml --profile tools run --rm scenario -- -scenario internal/test/game/scenarios/systems/daggerheart/basic_flow.lua
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml --profile tools run --rm maintenance -- replay -campaign-id <id> -validate
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml --profile tools run --rm catalog-importer
```

The `seed` tool in this repository is intentionally scoped to the local-dev manifest for local workflows. For any production-like environment, use a separate migration/administrative flow instead of `seed`.

## Volumes

Compose uses named volumes for SQLite data stores. To remove them:

```sh
docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml down -v
```
