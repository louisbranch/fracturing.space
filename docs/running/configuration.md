# Configuration

## Environment variables

- `FRACTURING_SPACE_GAME_EVENTS_DB_PATH`: file path for the event journal SQLite database. Default: `data/game-events.db`.
- `FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH`: file path for the projections SQLite database. Default: `data/game-projections.db`.
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY`: root secret used to sign event chain hashes. Required.
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEYS`: optional comma-separated key ring (`key_id=secret`).
- `FRACTURING_SPACE_GAME_EVENT_HMAC_KEY_ID`: active key id for signing when using the key ring. Default: `v1`.
- `FRACTURING_SPACE_GAME_ADDR`: default game server address used by the MCP server (when `-addr` is not set) and the admin dashboard (when `-grpc-addr` is not set). Defaults to `localhost:8080`.
- `FRACTURING_SPACE_ADMIN_ADDR`: HTTP bind address for the admin dashboard when `-http-addr` is not set. Defaults to `:8082`.
- `FRACTURING_SPACE_MCP_ALLOWED_HOSTS`: comma-separated list of allowed Host/Origin values for MCP HTTP transport. Defaults to loopback-only when unset.
- `FRACTURING_SPACE_MCP_HTTP_ADDR`: HTTP bind address for MCP transport when using HTTP and `-http-addr` is not set. Defaults to `0.0.0.0:8081`.

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

### Address Overrides

The admin dashboard accepts flags for gRPC and HTTP addresses. If `FRACTURING_SPACE_ADMIN_ADDR`
or `FRACTURING_SPACE_GAME_ADDR` are set, they provide defaults when the matching flag is
omitted. Command-line flags take precedence over env values.
