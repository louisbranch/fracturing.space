# Getting Started

## Prerequisites

- Go 1.25.6
- protoc (required until binaries are published)
- SQLite (embedded; the server creates `data/fracturing.space.db` by default)
- Make (for `make run`)

## Run locally (fastest)

Start the gRPC server, MCP bridge, and web client together:

```sh
make run
```

This runs the gRPC server on `localhost:8080`, the MCP server on stdio, and the web client on `http://localhost:8082`.
The MCP server will wait for the gRPC server to be healthy before accepting requests.

## Run services individually

Start the gRPC server:

```sh
go run ./cmd/server
```

Start the MCP server after the gRPC server starts.

```sh
go run ./cmd/mcp
```

Default endpoints:

- gRPC: `localhost:8080`
- MCP (stdio): process stdin/stdout
- Web: `http://localhost:8082`

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

Run with Compose (MCP HTTP on loopback, gRPC internal-only):

```sh
docker compose up
```

Compose uses a named volume for the gRPC data store. To remove it:

```sh
docker compose down -v
```

On first run, Compose initializes the volume permissions so the nonroot gRPC
container can write the database.

Check MCP health:

```sh
curl http://localhost:8081/mcp/health
```

## Docker (Remote deployment)

For remote deployments, keep MCP bound to loopback and front it with a reverse
proxy (Caddy/Nginx) that terminates TLS. Allow only your domain in
`FRACTURING_SPACE_MCP_ALLOWED_HOSTS`.

You can set `FRACTURING_SPACE_GRPC_ADDR` and `FRACTURING_SPACE_MCP_HTTP_ADDR` in the MCP container
instead of flags. Command-line flags still take precedence when provided.

Example (replace `your-domain.example`):

```sh
docker network create fracturing-space

docker run -d --name fracturing-space-grpc \
  --network fracturing-space \
  -p 127.0.0.1:8080:8080 \
  -v /srv/fracturing-space/data:/data \
  -e FRACTURING_SPACE_DB_PATH=/data/fracturing.space.db \
  docker.io/louisbranch/fracturing.space-grpc:latest

docker run -d --name fracturing-space-mcp \
  --network fracturing-space \
  -p 127.0.0.1:8081:8081 \
  -e FRACTURING_SPACE_MCP_ALLOWED_HOSTS=your-domain.example \
  docker.io/louisbranch/fracturing.space-mcp:latest \
  -transport=http -http-addr=0.0.0.0:8081 -addr=fracturing-space-grpc:8080

docker run -d --name fracturing-space-web \
  --network fracturing-space \
  -p 127.0.0.1:8082:8082 \
  -e FRACTURING_SPACE_WEB_ADDR=0.0.0.0:8082 \
  -e FRACTURING_SPACE_GRPC_ADDR=fracturing-space-grpc:8080 \
  docker.io/louisbranch/fracturing.space-web:latest
```

## Docker (Publish images)

Use bake to build and push all images:

```sh
GRPC_IMAGE="docker.io/louisbranch/fracturing.space-grpc:latest" \
MCP_IMAGE="docker.io/louisbranch/fracturing.space-mcp:latest" \
WEB_IMAGE="docker.io/louisbranch/fracturing.space-web:latest" \
docker buildx bake --push
```
