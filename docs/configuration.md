## Environment variables

- `DUALITY_DB_PATH`: file path for the campaign BoltDB store. Default: `data/duality.db`.
- `DUALITY_GRPC_ADDR`: gRPC address used by the MCP server. Defaults to `localhost:8080`.
- `DUALITY_MCP_ALLOWED_HOSTS`: comma-separated list of allowed Host/Origin values for MCP HTTP transport. Defaults to loopback-only when unset.
- `DUALITY_MCP_HTTP_ADDR`: HTTP bind address for MCP transport when running via the container entrypoint. Defaults to `0.0.0.0:8081`.

## MCP Server Configuration

### Command-line Flags

The MCP server (`cmd/mcp`) accepts the following flags:

- `-addr`: gRPC server address. Default: `localhost:8080`
- `-http-addr`: HTTP server address (for HTTP transport). Default: `localhost:8081`
  
  When running the `cmd/mcp` binary, this value is provided by the flag definition. When constructing the MCP server programmatically and leaving the HTTP address empty in the `Config` struct, the server also falls back to `localhost:8081` internally.
- `-transport`: Transport type (`stdio` or `http`). Default: `stdio`

### Address Overrides

The MCP server accepts a flag for the gRPC address. If `DUALITY_GRPC_ADDR`
is set, it takes precedence over the flag value.

### Transport Selection

The MCP server supports `stdio` (default) and `http` transports. See
[Getting started](getting-started.md) for run commands and
[MCP tools and resources](mcp.md) for HTTP endpoint details.
