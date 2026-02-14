# Getting Started

## Prerequisites

- Go 1.25.6
- protoc (required until binaries are published)
- SQLite (embedded; the server creates `data/game-events.db` and `data/game-projections.db` by default)
- Make (for `make run`)

## Run locally (fastest)

Start the game server, auth service, MCP bridge, and admin dashboard together
(the web login server runs separately; see Docker + Caddy for the full stack):

```sh
make run
```

This runs the game server on `localhost:8080`, the auth server on `localhost:8083`, the MCP server on stdio, and the admin dashboard on `http://localhost:8082`.
`make run` sets a local-only HMAC key (`dev-secret`) for the game server. Override it by exporting `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY`.
The MCP server will wait for the game server to be healthy before accepting requests.

## Run services individually

Start the game server:

```sh
FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=dev-secret \
go run ./cmd/game
```

Generate a secure HMAC key:

```sh
go run ./cmd/hmac-key
```

Generate a join-grant keypair:

```sh
go run ./cmd/join-grant-key
```

Start the auth server:

```sh
go run ./cmd/auth
```

Start the web login server:

```sh
go run ./cmd/web
```

Start the MCP server after the game server starts.

```sh
go run ./cmd/mcp
```

Default endpoints:

- Game gRPC: `localhost:8080`
- Auth gRPC: `localhost:8083`
- Auth HTTP: `http://localhost:8084`
- MCP (stdio): process stdin/stdout
- Admin: `http://localhost:8082`
- Web login: `http://localhost:8086/login`

## MCP HTTP transport (local only)

If you need the MCP bridge over HTTP for local tooling:

```sh
go run ./cmd/mcp -transport=http -http-addr=localhost:8081 -addr=localhost:8080
```

Default HTTP endpoint: `http://localhost:8081/mcp`

## Docker + Caddy (Local)

Copy the env template, generate keys, and start the full stack:

```sh
cp .env.example .env
go run ./cmd/hmac-key
go run ./cmd/join-grant-key
# Update .env with the generated values.
docker compose up --build
```

Paste the generated values into `.env` (the output is already in `.env` format).
The default `dev-secret` is only for local
exploration; replace it for any real data.

Caddy listens on `http://localhost:8080` by default (HTTPS on `https://localhost:8443` when enabled).
Local routes:

- Web: `http://localhost:8080/login`
- Auth: `http://auth.localhost:8080`
- Admin: `http://admin.localhost:8080`
- MCP health: `http://mcp.localhost:8080/mcp/health`

If you change the Caddy HTTP port, update `FRACTURING_SPACE_PUBLIC_PORT` to match.

Compose uses named volumes for data stores. To remove them:

```sh
docker compose down -v
```

On first run, Compose initializes the volume permissions so the nonroot
containers can write the databases.

## Docker + Caddy (Production)

1. Copy `.env.example` to `.env` and update the required values:

   - `FRACTURING_SPACE_DOMAIN=your-domain.example`
   - `FRACTURING_SPACE_PUBLIC_SCHEME=https`
   - `FRACTURING_SPACE_PUBLIC_PORT=` (empty)
   - `FRACTURING_SPACE_BIND_ADDR=0.0.0.0`
   - `FRACTURING_SPACE_HTTP_PORT=80`
   - `FRACTURING_SPACE_HTTPS_PORT=443`
   - `FRACTURING_SPACE_CADDY_AUTO_HTTPS=on`
   - `FRACTURING_SPACE_CADDY_SITE_PREFIX=`
   - `FRACTURING_SPACE_CADDY_EMAIL="email ops@your-domain.example"`
   - `FRACTURING_SPACE_IMAGE_TAG=latest`
   - `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=change-me`
   - `FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY=...`
   - `FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY=...`
   - `FRACTURING_SPACE_MCP_ALLOWED_HOSTS=mcp.your-domain.example`

2. Generate production keys and set them in `.env`:

   ```sh
   go run ./cmd/hmac-key
   go run ./cmd/join-grant-key
   ```

3. Pull and run:

   ```sh
   docker compose pull
   docker compose up -d
   ```

## Docker (Publish images)

Use bake to build and push all images:

```sh
GAME_IMAGE="ghcr.io/fracturing-space/game:latest" \
MCP_IMAGE="ghcr.io/fracturing-space/mcp:latest" \
ADMIN_IMAGE="ghcr.io/fracturing-space/admin:latest" \
AUTH_IMAGE="ghcr.io/fracturing-space/auth:latest" \
WEB_IMAGE="ghcr.io/fracturing-space/web:latest" \
docker buildx bake --push
```
