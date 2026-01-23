# Configuration

## Environment variables

- `DUALITY_DB_PATH`: file path for the campaign BoltDB store. Default: `data/duality.db`.
- `DUALITY_GRPC_ADDR`: gRPC address used by the MCP server. Defaults to `localhost:8080`.

## MCP address overrides

The MCP server accepts a flag for the gRPC address:

```sh
go run ./cmd/mcp -addr localhost:8080
```

If `DUALITY_GRPC_ADDR` is set, it takes precedence over the flag value.
