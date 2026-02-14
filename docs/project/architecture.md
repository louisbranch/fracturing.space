# Architecture

## Overview

Fracturing.Space is split into four layers:

- **Transport layer**: Game server (`cmd/game`) + Auth server (`cmd/auth`) + MCP bridge (`cmd/mcp`) + Admin dashboard (`cmd/admin`)
- **Platform layer**: Shared infrastructure (`internal/platform/`)
- **Domain layer**: Game systems (`internal/services/game/domain/systems/`) + Campaign model (`internal/services/game/domain/campaign/`)
- **Storage layer**: SQLite persistence (`data/game-events.db`, `data/game-projections.db`, `data/auth.db`, `data/admin.db`)

The MCP server is a thin adapter that forwards requests to the game services.
All rule evaluation and state changes live in the game server and domain
packages.

Service boundaries are organized under `internal/services/`. A service owns its
transport surface, application orchestration, domain logic, and storage adapters.
Shared utilities that do not expose an API surface (for example, RNG seed
generation) live in the domain or platform layers instead of being separate
services.

For domain terminology, see [domain-language.md](domain-language.md).

## Game System Architecture

Fracturing.Space supports multiple tabletop RPG systems through a pluggable architecture. Each game system is a plugin under `internal/services/game/domain/systems/`:

```
internal/services/game/domain/systems/
├── registry.go          # GameSystem interface + registration
└── daggerheart/         # Daggerheart implementation
    ├── domain/          # Duality dice, outcomes, probability
    ├── state.go         # System-specific state handlers
    └── content/         # Compendium and starter kit data (stub)
```

Game system gRPC services live in `internal/services/game/api/grpc/systems/{name}/`.

Systems are registered at startup and campaigns are bound to one system at creation.

For detailed information on the game system architecture, including how to add new systems, see [game-systems.md](game-systems.md).

## Campaign Model

Campaign data is organized into three tiers by change frequency:

| Layer | Subpackages | Changes | Contents |
|-------|-------------|---------|----------|
| **Campaign** (Config) | `campaign/`, `campaign/participant/`, `campaign/character/` | Setup time | Name, system, GM mode, participants, character profiles |
| **Snapshot** | `campaign/snapshot/` | At any event sequence | Materialized projection cache for replay/performance |
| **Session** (Gameplay) | `campaign/session/` | Every action | Active session, events, rolls, outcomes |

This model uses an event-sourced architecture where the event journal is the
source of truth and projections/snapshots are derived views.

## Event-Sourced Model

- All game changes are events in the campaign journal.
- Projections are derived through an explicit apply pipeline.
- Snapshots are derived for performance and replay acceleration.
- Only event emitters append to `game-events`; only appliers/adapters write to `game-projections`.
- Session is a query label; session events are not a separate journal.
- Story changes are first-class events in the same journal.
- Telemetry is stored separately from the game event log.

## High-level flow

```
Client (gRPC)            Client (MCP stdio/HTTP)
      |                            |
      | gRPC requests              | JSON-RPC requests
      v                            v
  Game server <----------------- MCP bridge
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

### Game server (gRPC)

The game server hosts the canonical API surface for rules and campaign state.
It validates inputs, applies the ruleset, and persists state.

Entry point: `cmd/game`

### Auth server

The auth server hosts the auth gRPC API for identity data (users) and future
authentication flows.

Entry point: `cmd/auth`

### MCP bridge

The MCP server exposes the same capabilities over the MCP JSON-RPC protocol.
It maintains per-client context for convenience, but does not own rules or
state logic.

Entry point: `cmd/mcp`

### Domain packages

Game system mechanics live in `internal/services/game/domain/systems/` (e.g., `internal/services/game/domain/systems/daggerheart/`).
The campaign model lives in `internal/services/game/domain/campaign/` with subpackages for participant,
character, snapshot, and session. Core primitives (dice, checks, RNG) live in
`internal/services/game/domain/core/`. These packages are transport agnostic and used by the game services.

### Storage

Persistent state is stored in SQLite (`modernc.org/sqlite`) with separate
databases per service boundary:

- Game service events: `data/game-events.db` (`FRACTURING_SPACE_GAME_EVENTS_DB_PATH`)
- Game service projections: `data/game-projections.db` (`FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH`)
- Auth service: `data/auth.db` (`FRACTURING_SPACE_AUTH_DB_PATH`)
- Admin service: `data/admin.db` (`FRACTURING_SPACE_ADMIN_DB_PATH`)

Each database is accessed only by its owning service and is not shared across
processes except through the service APIs.

## Services and boundaries

The primary service boundaries are:

- **Game service** (`internal/services/game/`): Canonical rules and campaign state; gRPC APIs under `internal/services/game/api/grpc/`; owns the game database.
- **MCP service** (`internal/services/mcp/`): JSON-RPC adapter for the MCP protocol; forwards to the game service and does not own rules or state.
- **Admin service** (`internal/services/admin/`): HTTP admin dashboard; renders UI and calls the game service for data.
- **Auth service** (`internal/services/auth/`): Authentication domain logic and gRPC API surface; owns the auth database.

Non-service utilities live in shared layers:

- **RNG/seed generation**: `internal/services/game/domain/core/random/` (shared domain utility, not a service).
- **Seeding CLI**: `cmd/seed` (dev tooling that calls the game service APIs).
