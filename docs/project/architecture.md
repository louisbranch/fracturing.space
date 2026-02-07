# Architecture

## Overview

Fracturing.Space is split into three layers:

- **Transport layer**: gRPC server (`cmd/server`) + MCP bridge (`cmd/mcp`) + Web UI (`cmd/web`)
- **Domain layer**: Game systems (`internal/systems/`) + State management (`internal/state/`)
- **Storage layer**: SQLite persistence (`data/fracturing.space.db`)

The MCP server is a thin adapter that forwards requests to the gRPC services.
All rule evaluation and state changes live in the gRPC server and domain
packages.

For domain terminology, see [domain-language.md](domain-language.md).

## Game System Architecture

Fracturing.Space supports multiple tabletop RPG systems through a pluggable architecture. Each game system is a plugin under `internal/systems/`:

```
internal/systems/
├── registry.go          # GameSystem interface + registration
└── daggerheart/         # Daggerheart implementation
    ├── domain/          # Duality dice, outcomes, probability
    ├── state.go         # System-specific state handlers
    └── content/         # Compendium and starter kit data (stub)
```

Game system gRPC services live in `internal/api/grpc/systems/{name}/`.

Systems are registered at startup and campaigns are bound to one system at creation.

For detailed information on the game system architecture, including how to add new systems, see [game-systems.md](game-systems.md).

## State Management Model

Game state is organized into three tiers by change frequency:

| Layer | Subpackages | Changes | Contents |
|-------|-------------|---------|----------|
| **Campaign** (Config) | `state/campaign/`, `state/participant/`, `state/character/` | Setup time | Name, system, GM mode, participants, character profiles |
| **Snapshot** (Continuity) | `state/snapshot/` | Between sessions | Character state (HP, Hope, Stress), GM Fear, progress |
| **Session** (Gameplay) | `state/session/` | Every action | Active session, events, rolls, outcomes |

This model uses an event-sourced architecture where the event journal is the
source of truth and projections/snapshots are derived views.

## Event-Sourced Model

- All game changes are events in the campaign journal.
- Projections are derived through an explicit apply pipeline.
- Snapshots are derived for performance and replay acceleration.
- Session is a query label; session events are not a separate journal.
- Story changes are first-class events in the same journal.
- Telemetry is stored separately from the game event log.

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
    SQLite
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

Game system mechanics live in `internal/systems/` (e.g., `internal/systems/daggerheart/`).
State management lives in `internal/state/` with subpackages for campaign, participant,
character, snapshot, and session. Core primitives (dice, checks, RNG) live in
`internal/core/`. These packages are transport agnostic and used by the gRPC services.

### Storage

Persistent state is stored in SQLite (`modernc.org/sqlite`) with a default path
of `data/fracturing.space.db`. The database is accessed by the server and is not shared
across processes except through the server APIs.
