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

The service also exposes a deterministic duality outcome evaluator that accepts
known Hope/Fear dice and returns the same structured outcome without rolling.

Mechanics at a glance: an action roll totals Hope + Fear + modifier. Matching
Hope and Fear is always a critical success and overrides difficulty checks.

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

- `duality_action_roll`: rolls Duality dice and returns the outcome with the roll context.
- `duality_outcome`: evaluates a deterministic outcome from known Hope/Fear dice without rolling.
- `roll_dice`: rolls arbitrary dice pools and returns the individual results.

### Example tool call (JSON-RPC over stdio)

Call the tool with `method: tools/call`, name `duality_action_roll`, and arguments
`modifier: 2` and `difficulty: 10`.

Example response payload (structured content fields):

`hope`, `fear`, `modifier`, `difficulty` (if provided), `total`, `is_crit`,
`meets_difficulty`, and `outcome`.

### OpenCode local MCP config (JSONC)

See `opencode.jsonc` for a ready-to-use OpenCode configuration that starts the
local MCP server.
