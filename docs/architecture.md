# Architecture

## Overview

Fracturing.Space is split into three layers:

- Transport layer: gRPC server and MCP bridge
- Domain layer: rules adjudication and campaign/session logic
- Storage layer: BoltDB-backed persistence

The MCP server is a thin adapter that forwards requests to the gRPC services.
All rule evaluation and state changes live in the gRPC server and domain
packages.

## High-level flow

```
Client (gRPC)            Client (MCP stdio/HTTP)
      |                            |
      | gRPC requests              | JSON-RPC requests
      v                            v
  gRPC server <----------------- MCP bridge
      |
      | domain service calls
      v
  Rules + Campaign/Session logic
      |
      | persistence
      v
    BoltDB
```

## Components

### gRPC server

The gRPC server hosts the canonical API surface for rules and campaign state.
It validates inputs, applies the ruleset, and persists state.

Entry point: `cmd/server`

### MCP bridge

The MCP server exposes the same capabilities over the MCP JSON-RPC protocol.
It maintains per-client context for convenience, but does not own rules or
state logic.

Entry point: `cmd/mcp`

### Domain packages

Rules and campaign/session behavior live in `internal/duality`,
`internal/campaign`, and `internal/session`. These packages are transport
agnostic and used by the gRPC services.

### Storage

Persistent state is stored in BoltDB (`go.etcd.io/bbolt`) with a default path
of `data/duality.db`. The database is accessed by the server and is not shared
across processes except through the server APIs.
