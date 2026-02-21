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
- First-run network access in devcontainer to download Go modules and install `air`.
- Option B (host-only): Go 1.26.0, protoc (until binaries are published), and Make

## Option A: Devcontainer (recommended)

```sh
# in VS Code
Reopen in Container
```

The devcontainer setup is defined in `.devcontainer/devcontainer.json` and starts a
watch-based runtime automatically after attach.

- `postCreateCommand` installs `air` (live-reload watcher for Go).
- `postStartCommand` launches `.devcontainer/scripts/watch-services.sh`.
- The watcher script initializes `.env` from `.env.local.example` when missing.
- The watcher script also generates join-grant keys when they are missing.

No manual `make run` or repeated `docker compose up` is needed for day-to-day edits.
Each restart still compiles, but only through Go build cache and only when files change.

Watcher controls:

```sh
make up    # start watchers (or re-start if needed)
make down  # stop watchers
```

Watcher logs:

- `.tmp/dev/game.log`
- `.tmp/dev/auth.log`
- `.tmp/dev/mcp.log`
- `.tmp/dev/admin.log`
- `.tmp/dev/web.log`
- `.tmp/dev/watch-services.log`

Stop watchers:

```sh
make down
```

## Option B: Host machine (existing flow)

```sh
make run
```

`make run` reads environment variables from `.env`; if `.env` does not exist, it is
initialized from the file specified by `$ENV_EXAMPLE` (defaulting to `.env.local.example`).

`make run` starts the game server, auth service, MCP bridge, and admin dashboard.
It also generates dev join-grant keys if they are missing.

## Default endpoints

- Game gRPC: `localhost:8080`
- Auth gRPC: `localhost:8083`
- Auth HTTP: `http://localhost:8084`
- MCP (stdio): process stdin/stdout
- Admin: `http://localhost:8082`
- Web login: `http://localhost:8086/login`

## Demo data

See [seeding](seeding.md) for `make seed` and generator options.

## Configuration

See [configuration](configuration.md) for the full environment variable reference.

For external image hosting (recommended for campaign covers and avatars), set:

- `FRACTURING_SPACE_ASSET_BASE_URL` (for example `https://cdn.example.com/assets`)
- `FRACTURING_SPACE_ASSET_VERSION` (for example `v1`)

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
