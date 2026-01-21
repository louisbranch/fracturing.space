# Duality Engine

Duality Engine is a small, server-authoritative mechanics service for Daggerheart-style Duality Dice resolution.

It is not a narrative engine.
It does not generate lore, scenes, or roleplay.
It provides explicit, auditable mechanical outcomes via a gRPC API.

## What it does

Duality Engine exposes a gRPC service that resolves "action rolls" using Duality Dice:

- roll Hope d12 and Fear d12
- compute totals with a modifier
- optionally compare against a difficulty
- return structured output (dice, total, outcome)

Clients can be:
- an MCP bridge for LLM tool calls
- a web UI for humans
- anything else that can call gRPC

## MCP (stdio)

The MCP server communicates over stdio using JSON-RPC. Run it locally and point
your MCP client at the process stdin/stdout.

Run the MCP server (defaults to gRPC at localhost:8080):

```sh
go run ./cmd/mcp
```

Override the gRPC target:

```sh
go run ./cmd/mcp -addr localhost:8080
```

Or with an environment variable:

```sh
DUALITY_GRPC_ADDR=localhost:8080 go run ./cmd/mcp
```

### Available tools

- `duality_action_roll`: rolls Duality dice for an action.
  - inputs: `modifier` (number, default 0), `difficulty` (number, optional)
  - outputs: `hope`, `fear`, `total`, `modifier`, `outcome`, `difficulty` (optional)

### Example tool call (JSON-RPC over stdio)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "duality_action_roll",
    "arguments": {
      "modifier": 2,
      "difficulty": 10
    }
  }
}
```

Example response payload:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{\"hope\":4,\"fear\":7,\"total\":13,\"modifier\":2,\"outcome\":\"SUCCESS_WITH_HOPE\",\"difficulty\":10}"
      }
    ],
    "structuredContent": {
      "hope": 4,
      "fear": 7,
      "total": 13,
      "modifier": 2,
      "outcome": "SUCCESS_WITH_HOPE",
      "difficulty": 10
    }
  }
}
```

### OpenCode local MCP config (JSONC)

See `opencode.jsonc` for a ready-to-use OpenCode configuration that starts the
local MCP server.
