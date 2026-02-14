# Configuration

## Environment variables

### Game

- `FRACTURING_SPACE_GAME_EVENTS_DB_PATH`: event journal SQLite path. Default: `data/game-events.db`.
- `FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH`: projections SQLite path. Default: `data/game-projections.db`.
- `FRACTURING_SPACE_GAME_CONTENT_DB_PATH`: content SQLite path. Default: `data/game-content.db`.
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY`: root secret used to sign event chain hashes. Required. Generate with `go run ./cmd/hmac-key`.
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS`: optional comma-separated key ring (`key_id=secret`).
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID`: active key id when using the key ring. Default: `v1`.

### Auth + OAuth

- `FRACTURING_SPACE_AUTH_DB_PATH`: auth SQLite path. Default: `data/auth.db`.
- `FRACTURING_SPACE_AUTH_PORT`: gRPC port for auth service. Default: `8083`.
- `FRACTURING_SPACE_AUTH_HTTP_ADDR`: HTTP bind address for OAuth endpoints. Default: `localhost:8084`.
- `FRACTURING_SPACE_OAUTH_ISSUER`: external OAuth issuer URL. Defaults to the auth HTTP address when unset.
- `FRACTURING_SPACE_OAUTH_LOGIN_UI_URL`: external login UI URL for redirects (web login server).
- `FRACTURING_SPACE_OAUTH_LOGIN_REDIRECTS`: comma-separated list of allowed login redirect URLs.
- `FRACTURING_SPACE_OAUTH_CLIENTS`: JSON array of registered OAuth clients.
- `FRACTURING_SPACE_OAUTH_USERS`: JSON array of bootstrap users.
- `FRACTURING_SPACE_OAUTH_RESOURCE_SECRET`: shared secret for resource introspection.

### Join grants

- `FRACTURING_SPACE_JOIN_GRANT_ISSUER`: join grant issuer.
- `FRACTURING_SPACE_JOIN_GRANT_AUDIENCE`: join grant audience.
- `FRACTURING_SPACE_JOIN_GRANT_PUBLIC_KEY`: base64 Ed25519 public key for verification (game). Generate with `go run ./cmd/join-grant-key`.
- `FRACTURING_SPACE_JOIN_GRANT_PRIVATE_KEY`: base64 Ed25519 private key for signing (auth). Generate with `go run ./cmd/join-grant-key`.
- `FRACTURING_SPACE_JOIN_GRANT_TTL`: join grant TTL. Default: `5m`.

### MCP

- `FRACTURING_SPACE_GAME_ADDR`: game gRPC address used by MCP and admin. Default: `localhost:8080`.
- `FRACTURING_SPACE_MCP_HTTP_ADDR`: HTTP bind address for MCP when using HTTP transport. Default: `localhost:8081`.
- `FRACTURING_SPACE_MCP_TRANSPORT`: transport type (`stdio` or `http`). Default: `stdio`.
- `FRACTURING_SPACE_MCP_ALLOWED_HOSTS`: comma-separated allowed Host/Origin values for MCP HTTP. Defaults to loopback-only when unset.
- `FRACTURING_SPACE_MCP_OAUTH_ISSUER`: OAuth issuer URL for MCP token validation.
- `FRACTURING_SPACE_MCP_OAUTH_RESOURCE_SECRET`: shared secret for MCP token introspection.

### Admin

- `FRACTURING_SPACE_ADMIN_ADDR`: HTTP bind address for the admin dashboard. Default: `:8082`.
- `FRACTURING_SPACE_ADMIN_DB_PATH`: admin SQLite path. Default: `data/admin.db`.
- `FRACTURING_SPACE_AUTH_ADDR`: auth gRPC address used by the game, admin dashboard, and web login server.

### Web

- `FRACTURING_SPACE_WEB_HTTP_ADDR`: HTTP bind address for the web login server. Default: `localhost:8086`.
- `FRACTURING_SPACE_WEB_AUTH_BASE_URL`: external auth base URL for login redirects.
- `FRACTURING_SPACE_WEB_AUTH_ADDR`: auth gRPC address used by the web login server.
- `FRACTURING_SPACE_WEB_DIAL_TIMEOUT`: gRPC dial timeout for the web login server. Default: `2s`.

### Docker + Caddy (Compose defaults)

- `FRACTURING_SPACE_DOMAIN`: base domain for subdomain routing (e.g., `example.com`).
- `FRACTURING_SPACE_PUBLIC_SCHEME`: external scheme for URLs (`http` or `https`).
- `FRACTURING_SPACE_PUBLIC_PORT`: optional port suffix (include the leading `:`; e.g., `:8080` for local).
- `FRACTURING_SPACE_BIND_ADDR`: bind address for Caddy ports (local defaults to `127.0.0.1`).
- `FRACTURING_SPACE_HTTP_PORT`: host port mapped to Caddy HTTP (default `8080` for local).
- `FRACTURING_SPACE_HTTPS_PORT`: host port mapped to Caddy HTTPS (default `8443` for local).
- `FRACTURING_SPACE_CADDY_AUTO_HTTPS`: Caddy `auto_https` setting (`off` for local, `on` for prod).
- `FRACTURING_SPACE_CADDY_SITE_PREFIX`: Caddy site prefix (`http://` for local, empty for production TLS).
- `FRACTURING_SPACE_CADDY_EMAIL`: optional Caddy global directive (e.g., `email ops@example.com`).

### Images

- `FRACTURING_SPACE_IMAGE_REGISTRY`: container registry host. Default: `ghcr.io`.
- `FRACTURING_SPACE_IMAGE_NAMESPACE`: container namespace/org. Default: `fracturing-space`.
- `FRACTURING_SPACE_IMAGE_TAG`: image tag used in Compose. Default: `dev`.

## MCP Server Configuration

### Command-line Flags

The MCP server (`cmd/mcp`) accepts the following flags:

- `-addr`: game server address. Default: `localhost:8080`
- `-http-addr`: HTTP server address (for HTTP transport). Default: `localhost:8081`
  
  When running the `cmd/mcp` binary, this value is provided by the flag definition. When constructing the MCP server programmatically and leaving the HTTP address empty in the `Config` struct, the server also falls back to `localhost:8081` internally.
- `-transport`: Transport type (`stdio` or `http`). Default: `stdio`

### Address Overrides

The MCP server accepts flags for gRPC and HTTP addresses. If `FRACTURING_SPACE_GAME_ADDR`
or `FRACTURING_SPACE_MCP_HTTP_ADDR` are set, they provide defaults when the matching flag
is omitted. Command-line flags take precedence over env values.

### Transport Selection

The MCP server supports `stdio` (default) and `http` transports. See
[Getting started](getting-started.md) for run commands and
[MCP tools and resources](../reference/mcp.md) for HTTP endpoint details.

## Admin Dashboard Configuration

### Command-line Flags

The admin dashboard (`cmd/admin`) accepts the following flags:

- `-http-addr`: HTTP server address. Default: `:8082`
- `-grpc-addr`: game server address. Default: `localhost:8080`
- `-auth-addr`: auth server address. Default: `localhost:8083`

### Address Overrides

The admin dashboard accepts flags for gRPC and HTTP addresses. If `FRACTURING_SPACE_ADMIN_ADDR`
or `FRACTURING_SPACE_GAME_ADDR` are set, they provide defaults when the matching flag is
omitted. Command-line flags take precedence over env values.

## Web Login Configuration

### Command-line Flags

The web login server (`cmd/web`) accepts the following flags:

- `-http-addr`: HTTP server address. Default: `localhost:8086`
- `-auth-base-url`: external auth base URL used in login redirects.
- `-auth-addr`: auth server address. Default: `localhost:8083`

### Address Overrides

The web login server accepts flags for HTTP address and auth endpoints. If
`FRACTURING_SPACE_WEB_HTTP_ADDR`, `FRACTURING_SPACE_WEB_AUTH_BASE_URL`, or
`FRACTURING_SPACE_WEB_AUTH_ADDR` are set, they provide defaults when the matching flag
is omitted. Command-line flags take precedence over env values.
