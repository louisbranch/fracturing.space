# Duality Engine

Duality Engine is a small, server-authoritative mechanics service
compatible with the Daggerheart Duality Dice system.

It is not a narrative engine.
It does not generate lore, scenes, or roleplay.
It provides explicit, auditable mechanical outcomes via a gRPC API.

## What it does

Duality Engine exposes a gRPC service that resolves "action rolls" using Duality Dice:

The API is defined in `api/proto/duality/v1/duality.proto` and exposes the `DualityService`.

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

## Run locally

Start the gRPC server and MCP bridge together:

```sh
make run
```

This runs the gRPC server on `localhost:8080`, waits for it to accept
connections, and then starts the MCP server on stdio.

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
- `duality_explain`: returns a deterministic explanation for a known Hope/Fear outcome.
- `duality_probability`: computes exact outcome counts across all duality dice combinations.
- `duality_rules_version`: returns the ruleset semantics used for Duality roll evaluation.
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

---

## Attribution

This project uses material from the Daggerheart System Reference Document (SRD).

Daggerheart is © 2025 Critical Role LLC.
Used under the Darrington Press Community Gaming License.

## Disclaimer

Duality Engine is an independent, fan-made project.

It is not affiliated with, endorsed by, or sponsored by Critical Role LLC
or Darrington Press.

Daggerheart is a trademark of Critical Role LLC.

## SRD Content

This repository does not include Daggerheart SRD content.

Users are responsible for providing any SRD-derived data
in accordance with the Darrington Press Community Gaming License.
