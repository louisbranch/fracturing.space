# Getting Started

## Prerequisites

- Go 1.25.6
- protoc (required until binaries are published)
- SQLite (embedded; the server creates `data/game-events.db` and `data/game-projections.db` by default)
- Make (for `make run`)

## Run locally (fastest)

Start the game server, auth service, MCP bridge, and admin dashboard together:

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

Start the auth server:

```sh
go run ./cmd/auth
```

Start the MCP server after the game server starts.

```sh
go run ./cmd/mcp
```

Default endpoints:

- Game gRPC: `localhost:8080`
- Auth gRPC: `localhost:8083`
- MCP (stdio): process stdin/stdout
- Admin: `http://localhost:8082`

## MCP HTTP transport (local only)

If you need the MCP bridge over HTTP for local tooling:

```sh
go run ./cmd/mcp -transport=http -http-addr=localhost:8081 -addr=localhost:8080
```

Default HTTP endpoint: `http://localhost:8081/mcp`

## Docker (Local testing)

Build the images with bake:

```sh
docker buildx bake
```

Run with Compose (MCP HTTP on loopback, game/auth gRPC internal-only):

```sh
docker compose up
```

Compose uses a named volume for the game/auth data stores. To remove it:

```sh
docker compose down -v
```

On first run, Compose initializes the volume permissions so the nonroot
containers can write the databases.

Check MCP health:

```sh
curl http://localhost:8081/mcp/health
```

## Docker (Remote deployment)

For remote deployments, keep MCP bound to loopback and front it with a reverse
proxy (Caddy/Nginx) that terminates TLS. Allow only your domain in
`FRACTURING_SPACE_MCP_ALLOWED_HOSTS`.

You can set `FRACTURING_SPACE_GAME_ADDR` and `FRACTURING_SPACE_MCP_HTTP_ADDR` in the MCP container
instead of flags. Command-line flags still take precedence when provided.

Example (replace `your-domain.example`):

```sh
docker network create fracturing-space

	docker run -d --name fracturing-space-game \
	  --network fracturing-space \
	  -p 127.0.0.1:8080:8080 \
	  -v /srv/fracturing-space/data:/data \
	  -e FRACTURING_SPACE_GAME_EVENTS_DB_PATH=/data/game-events.db \
	  -e FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH=/data/game-projections.db \
	  -e FRACTURING_SPACE_GAME_EVENT_HMAC_KEY=change-me \
	  docker.io/louisbranch/fracturing.space-game:latest

docker run -d --name fracturing-space-auth \
  --network fracturing-space \
  -p 127.0.0.1:8083:8083 \
  -v /srv/fracturing-space/data:/data \
  -e FRACTURING_SPACE_AUTH_DB_PATH=/data/auth.db \
  docker.io/louisbranch/fracturing.space-auth:latest

docker run -d --name fracturing-space-mcp \
  --network fracturing-space \
  -p 127.0.0.1:8081:8081 \
  -e FRACTURING_SPACE_MCP_ALLOWED_HOSTS=your-domain.example \
  docker.io/louisbranch/fracturing.space-mcp:latest \
  -transport=http -http-addr=0.0.0.0:8081 -addr=fracturing-space-game:8080

docker run -d --name fracturing-space-admin \
  --network fracturing-space \
  -p 127.0.0.1:8082:8082 \
  -e FRACTURING_SPACE_ADMIN_ADDR=0.0.0.0:8082 \
  -e FRACTURING_SPACE_GAME_ADDR=fracturing-space-game:8080 \
  -e FRACTURING_SPACE_AUTH_ADDR=fracturing-space-auth:8083 \
  docker.io/louisbranch/fracturing.space-admin:latest
```

## Docker (Publish images)

Use bake to build and push all images:

```sh
GAME_IMAGE="docker.io/louisbranch/fracturing.space-game:latest" \
MCP_IMAGE="docker.io/louisbranch/fracturing.space-mcp:latest" \
ADMIN_IMAGE="docker.io/louisbranch/fracturing.space-admin:latest" \
AUTH_IMAGE="docker.io/louisbranch/fracturing.space-auth:latest" \
docker buildx bake --push
```
