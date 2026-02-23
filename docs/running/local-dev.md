---
title: "Local development"
parent: "Running"
nav_order: 2
---

# Local development (Go)

## Prerequisites

- Option A (devcontainer):
- Docker Engine/Desktop with Compose v2.
- Devcontainer-capable editor integration (for example, VS Code Dev Containers extension).
- First-run network access in devcontainer to download Go modules.
- Option B (host-only): Go 1.26.0, protoc (until binaries are published), and Make

## Option A: Devcontainer (recommended)

```sh
# in VS Code
Reopen in Container
```

The devcontainer setup is defined in `.devcontainer/devcontainer.json` and starts a
watch-based runtime automatically after attach.

- `postCreateCommand` verifies required watcher tooling is available.
- `postStartCommand` launches `.devcontainer/scripts/watch-services.sh`.
- The watcher script initializes `.env` from `.env.local.example` when missing.
- The watcher script also generates join-grant keys when they are missing.

The default watcher set starts `game`, `auth`, `connections`, `ai`,
`notifications`, `worker`, `mcp`, `admin`, `chat`, and `web`.

No manual multi-process `go run` orchestration or repeated `docker compose up` is needed for day-to-day edits.
Each restart still compiles, but only through Go build cache and only when files change.

Lifecycle controls:

```sh
make up    # start devcontainer + watchers (or re-start watchers if already inside container)
make down  # stop watchers + stop devcontainer (or just stop watchers if run inside container)
```

Ownership note:

- `make up` forwards your host UID/GID to compose (`DEVCONTAINER_UID`/`DEVCONTAINER_GID`) so files created under `.tmp/` and `data/` stay removable without `sudo`.
- Devcontainer `HOME` is `/home/vscode`, so tool state like `.config/go/telemetry` stays out of the workspace.
- Go module cache lives in container `/tmp/go-modcache` (not workspace), and `GOFLAGS=-modcacherw` is set in devcontainer flows.
- You can override explicitly before startup, for example: `DEVCONTAINER_UID=1001 DEVCONTAINER_GID=1001 make up`.
- If you already have stale root-owned artifacts from older runs, repair once with:

```sh
sudo chown -R "$(id -u):$(id -g)" .tmp data
```

Watcher logs:

- `.tmp/dev/game.log`
- `.tmp/dev/auth.log`
- `.tmp/dev/connections.log`
- `.tmp/dev/mcp.log`
- `.tmp/dev/admin.log`
- `.tmp/dev/chat.log`
- `.tmp/dev/ai.log`
- `.tmp/dev/notifications.log`
- `.tmp/dev/worker.log`
- `.tmp/dev/web.log`
- `.tmp/dev/watch-services.log`

Stop runtime:

```sh
make down
```

## Option B: Host machine (manual)

```sh
go run ./cmd/game
go run ./cmd/auth
go run ./cmd/connections
go run ./cmd/notifications
go run ./cmd/worker
go run ./cmd/mcp
go run ./cmd/admin
go run ./cmd/chat
go run ./cmd/ai
go run ./cmd/web
```

For host-only manual startup, initialize `.env` first (for example from `.env.local.example`)
and export join-grant keys when missing:

```sh
cp .env.local.example .env  # if .env does not exist yet
set -a
. ./.env
set +a
eval "$(go run ./cmd/join-grant-key)"
export FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY
```

If you run `cmd/ai`, set `FRACTURING_SPACE_AI_ENCRYPTION_KEY` to a base64-encoded
AES key (16/24/32 bytes) before startup.

## Default endpoints

- Game gRPC: `localhost:8082`
- Auth gRPC: `localhost:8083`
- Auth HTTP: `http://localhost:8084`
- Connections gRPC: `localhost:8090`
- MCP HTTP: `http://localhost:8085/mcp/health`
- Admin: `http://localhost:8081`
- Chat: `http://localhost:8086`
- AI gRPC: `localhost:8087`
- Notifications gRPC: `localhost:8088`
- Worker gRPC health: `localhost:8089`
- Web login: `http://localhost:8080/login`

## Demo data

See [seeding](seeding.md) for `make seed`.

## Configuration

See [configuration](configuration.md) for the canonical runtime variable reference.

For external image hosting (recommended for campaign covers and avatars), set:

- `FRACTURING_SPACE_ASSET_BASE_URL` (for example `https://cdn.example.com/assets`)
- `FRACTURING_SPACE_ASSET_VERSION` (for example `v1`)

If you already have a `.env` file, update those values there and restart watchers:

```sh
make down && make up
```

If you don't have a `.env` file yet:

```sh
cp .env.local.example .env
```

Upload helper:

```sh
scripts/upload-assets.sh \
  --source-dir ./assets/campaign-covers \
  --bucket-url s3://fracturing-space-assets \
  --version v1 \
  --domain campaign-covers \
  --set-id campaign_cover_set_v1 \
  --ext png
```
