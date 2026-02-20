---
title: "Local development"
parent: "Running"
nav_order: 2
---

# Local development (Go)

## Prerequisites

- Go 1.25.6
- protoc (until binaries are published)
- Make

## Run the core services

```sh
make run
```

`make run` starts the game server, auth service, MCP bridge, and admin dashboard.
It also generates dev join-grant keys if they are missing.

## Optional web login server

If you want the login UI without Docker:

```sh
go run ./cmd/web
```

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
