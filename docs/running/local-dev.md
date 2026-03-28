---
title: "Local development"
parent: "Running"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
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
- After game reaches its ready log marker, the watcher launches the catalog importer asynchronously.
- The catalog importer automatically retries transient SQLite busy/locked failures during startup.

The default watcher set starts `status`, `game`, `auth`, `social`, `discovery`,
`ai`, `notifications`, `userhub`, `worker`, `admin`, `play`,
and `web`.
Game reports catalog-backed capabilities as degraded until import completes, then
re-evaluates and promotes them to operational automatically.

No manual multi-process `go run` orchestration or repeated `docker compose up` is needed for day-to-day edits.
Each restart still compiles, but only through Go build cache and only when files change.

Lifecycle controls:

```sh
make up    # start devcontainer + watchers (or re-start watchers if already inside container)
make down  # stop watchers + stop devcontainer (or just stop watchers if run inside container)
```

When removing a worktree with `ofsht rm`, the repo-local delete hook now runs
`scripts/worktree-pre-delete.sh` first. That hook best-effort stops the
watcher/devcontainer stack for the target worktree and removes disposable local
state such as `data/` and `.tmp/` before the worktree is deleted. Use
`make down` when you want to stop the runtime without deleting the worktree.

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
- `.tmp/dev/social.log`
- `.tmp/dev/admin.log`
- `.tmp/dev/play.log`
- `.tmp/dev/ai.log`
- `.tmp/dev/notifications.log`
- `.tmp/dev/worker.log`
- `.tmp/dev/web.log`
- `.tmp/dev/storybook.log`
- `.tmp/dev/catalog-importer.log`
- `.tmp/dev/watch-services.log`

Stop runtime:

```sh
make down
```

## Option B: Host machine (manual)

```sh
go run ./cmd/game
go run ./cmd/auth
go run ./cmd/social
go run ./cmd/notifications
go run ./cmd/worker
go run ./cmd/admin
go run ./cmd/play
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
- Social gRPC: `localhost:8090`
- Admin: `http://localhost:8081`
- Play: `http://localhost:8094/up`
- AI gRPC: `localhost:8087`
- Notifications gRPC: `localhost:8088`
- Worker gRPC health: `localhost:8089`
- Web login: `http://localhost:8080/login`

In devcontainer watcher mode, the game handoff uses the direct play port
(`localhost:8094`) rather than `play.localhost:8080` because Caddy is not in
front of the watcher processes.

## Play frontend dev servers

`play` serves embedded assets by default. For bundled SPA-shell iteration, run
the frontend workspace directly and point `play` at the browser-reachable Vite
dev server origin:

```sh
cd internal/services/play/ui
npm ci
npm run dev -- --host 0.0.0.0 --port 5173
```

Set `FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL=http://localhost:5173` before
starting `cmd/play` or the devcontainer watchers. When unset, `play` serves the
checked-in build under `internal/services/play/ui/dist`.

In devcontainer watcher mode without `FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL`,
the `play` watcher now uses the checked-in embedded UI bundle as-is and does
not rebuild it. For live frontend iteration, run the Vite dev server and set
`FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL`. To refresh the committed embedded
bundle from the canonical Docker/Linux toolchain, run:

```sh
make play-ui-dist
```

In devcontainer mode, port `5173` is forwarded to the host so the browser can
reach the Vite server directly. After changing `.env` for
`FRACTURING_SPACE_PLAY_UI_DEV_SERVER_URL`, restart the watchers with
`make down && make up`.

For isolated component work, run Storybook separately:

```sh
cd internal/services/play/ui
npm ci
npm run storybook
```

Open:

```text
http://localhost:6006
```

`/` on the play service now shows a placeholder shell that points contributors
to Storybook, and `/preview/character-card` is retired.

In devcontainer watcher mode, `make up` now starts Storybook automatically and
forwards port `6006` to the host. `make down` stops it with the rest of the
watcher stack.

To verify the UI workspace from a clean checkout without rewriting the checked-in
bundle under `internal/services/play/ui/dist`, run:

```sh
make play-ui-check
```

That command installs the workspace dependencies, runs the Vitest suite, builds
the Vite bundle, and verifies the Storybook build in one place.

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
