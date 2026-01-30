# Getting Started

## Prerequisites

- Go 1.25.6
- protoc (required until binaries are published)
- BoltDB (embedded; the server creates `data/duality.db` by default)
- Make (for `make run`)

## Run locally

Start the gRPC server and MCP bridge together:

```sh
make run
```

This runs the gRPC server on `localhost:8080` and the MCP server on stdio.
The MCP server will wait for the gRPC server to be healthy before accepting requests.

## Run services individually

Start the gRPC server:

```sh
go run ./cmd/server
```

Start the MCP server after the gRPC server starts. See
[MCP tools and resources](mcp.md) for the MCP run command.

## Docker (Local testing)

Build the image:

```sh
docker build -t duality-engine:dev .
```

Or use the helper script (builds and runs):

```sh
./scripts/docker-run.sh
```

Create a local data directory for BoltDB:

```sh
mkdir -p data
sudo chown -R 65532:65532 data
```

Run the container (MCP HTTP on loopback, gRPC internal-only):

```sh
docker run \
  -p 127.0.0.1:8081:8081 \
  -v $(pwd)/data:/data \
  -e DUALITY_DB_PATH=/data/duality.db \
  -e DUALITY_GRPC_ADDR=127.0.0.1:8080 \
  -e DUALITY_MCP_ALLOWED_HOSTS=localhost \
  duality-engine:dev
```

Check MCP health:

```sh
curl http://localhost:8081/mcp/health
```

## Docker (Remote deployment)

For remote deployments, keep MCP bound to loopback and front it with a reverse
proxy (Caddy/Nginx) that terminates TLS. Allow only your domain in
`DUALITY_MCP_ALLOWED_HOSTS`.

Example (replace `your-domain.example`):

```sh
docker run \
  -p 127.0.0.1:8081:8081 \
  -v /srv/duality/data:/data \
  -e DUALITY_DB_PATH=/data/duality.db \
  -e DUALITY_GRPC_ADDR=127.0.0.1:8080 \
  -e DUALITY_MCP_ALLOWED_HOSTS=your-domain.example \
  duality-engine:dev
```
