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
