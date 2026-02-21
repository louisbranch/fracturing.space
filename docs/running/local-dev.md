---
title: "Local development"
parent: "Running"
nav_order: 2
---

# Local development (Go)

## Prerequisites

- Go 1.26.0
- protoc (until binaries are published)
- Make

## Run the core services

```sh
make run
```

`make run` reads environment variables from `.env`; if `.env` does not exist, it is
initialized from the file specified by `$ENV_EXAMPLE` (defaulting to `.env.local.example`).

`make run` starts the game server, auth service, MCP bridge, and admin dashboard.
It also generates dev join-grant keys if they are missing.

## Optional web login server

If you want the login UI without Docker:

```sh
go run ./cmd/web
```

If you run the web server in its own shell, set `FRACTURING_SPACE_WEB_OAUTH_CLIENT_ID`
and `FRACTURING_SPACE_WEB_CALLBACK_URL` consistently with your auth client settings.

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
