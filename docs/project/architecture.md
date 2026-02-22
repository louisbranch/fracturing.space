---
title: "Architecture"
parent: "Project"
nav_order: 2
---

# Architecture

## Overview

Fracturing.Space is split into four layers:

- **Transport layer**: Game server (`cmd/game`) + Auth server (`cmd/auth`) + Connections server (`cmd/connections`) + AI server (`cmd/ai`) + MCP bridge (`cmd/mcp`) + Admin dashboard (`cmd/admin`)
- **Platform layer**: Shared infrastructure (`internal/platform/`)
- **Domain layer**: Core domain packages (`internal/services/game/domain/{campaign,participant,character,invite,session,action,...}`) + game systems (`internal/services/game/domain/systems/`)
- **Storage layer**: SQLite persistence (`data/game-events.db`, `data/game-projections.db`, `data/game-content.db`, `data/auth.db`, `data/connections.db`, `data/admin.db`, `data/ai.db`)

The MCP server is a thin adapter that forwards requests to the game services.
All rule evaluation and state changes live in the game server and domain
packages.

Service boundaries are organized under `internal/services/`. A service owns its
transport surface, application orchestration, domain logic, and storage adapters.
Shared utilities that do not expose an API surface (for example, RNG seed
generation) live in the domain or platform layers instead of being separate
services.

For domain terminology, see [domain-language.md](domain-language.md).
For the canonical event-driven write model, see
[Event-driven system](event-driven-system.md).

## Game System Architecture

Fracturing.Space supports multiple tabletop RPG systems through a pluggable architecture. Each game system is a plugin under `internal/services/game/domain/systems/`:

```
internal/services/game/domain/systems/
├── adapter_registry.go  # Projection adapter routing by system + version
├── registry_bridge.go   # API-facing game-system metadata registry
└── daggerheart/         # Daggerheart implementation
    ├── module.go        # domain/system.Module implementation
    ├── decider.go       # system-owned command decisions
    ├── projector.go     # system-owned replay/projector logic
    ├── adapter.go       # projection adapter for system tables
    └── domain/          # pure mechanics (outcomes/probability/etc.)
```

Domain command/event module routing is defined in
`internal/services/game/domain/system/registry.go`.
Game-system gRPC surfaces live in
`internal/services/game/api/grpc/systems/{name}/`.
Systems are registered at startup, and campaigns are bound to one system at
creation.

For detailed information on the game system architecture, including how to add new systems, see [game-systems.md](game-systems.md).

## Campaign Model

Campaign data is organized into three tiers by change frequency:

| Layer | Packages | Changes | Contents |
|-------|-------------|---------|----------|
| **Core campaign state** | `campaign/`, `participant/`, `character/`, `invite/` | Setup + lifecycle | Name, status, seats, characters, invites |
| **Session gameplay** | `session/`, `action/` | During play | Active session, spotlight/gates, action resolution |
| **Derived projections** | `projection/`, `domain/systems/*/adapter.go` | Rebuilt/apply-time | Query models and system extension state |

This model uses an event-sourced architecture where the event journal is the
source of truth and projections/snapshots are derived views.

## Event-Sourced Model

Reference guide: [Event-driven system](event-driven-system.md)

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

### AI server

The AI server hosts provider credential and agent APIs for BYO AI access.
It owns provider secret lifecycle and AI agent configuration storage.

Entry point: `cmd/ai`

### Connections server

The connections server hosts user-directed contact APIs used for discovery and
invite targeting workflows.

Entry point: `cmd/connections`

### MCP bridge

The MCP server exposes the same capabilities over the MCP JSON-RPC protocol.
It maintains per-client context for convenience, but does not own rules or
state logic.

Entry point: `cmd/mcp`

### Domain packages

Game system mechanics live in `internal/services/game/domain/systems/` (e.g., `internal/services/game/domain/systems/daggerheart/`).
Core domain behavior lives in top-level domain packages such as
`internal/services/game/domain/campaign/`,
`internal/services/game/domain/participant/`,
`internal/services/game/domain/character/`,
`internal/services/game/domain/invite/`,
`internal/services/game/domain/session/`, and
`internal/services/game/domain/action/`.
Core primitives (dice, checks, RNG) live in `internal/services/game/core/`.
These packages are transport agnostic and used by the game services.

### Storage

Persistent state is stored in SQLite (`modernc.org/sqlite`) with separate
databases per service boundary:

- Game service events: `data/game-events.db` (`FRACTURING_SPACE_GAME_EVENTS_DB_PATH`)
- Game service projections: `data/game-projections.db` (`FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH`)
- Game service content catalog: `data/game-content.db` (`FRACTURING_SPACE_GAME_CONTENT_DB_PATH`)
- Auth service: `data/auth.db` (`FRACTURING_SPACE_AUTH_DB_PATH`)
- Connections service: `data/connections.db` (`FRACTURING_SPACE_CONNECTIONS_DB_PATH`)
- Admin service: `data/admin.db` (`FRACTURING_SPACE_ADMIN_DB_PATH`)
- AI service: `data/ai.db` (`FRACTURING_SPACE_AI_DB_PATH`)

Planned:

- Game service narratives: `data/game-narratives.db` (`FRACTURING_SPACE_GAME_NARRATIVES_DB_PATH`)

Each database is accessed only by its owning service and is not shared across
processes except through the service APIs.

The game service intentionally splits storage into distinct databases:

- Events (append-only journal).
- Projections (derived, mutable state).
- Content catalog (static, admin/import managed data).

## Services and boundaries

The primary service boundaries are:

- **Game service** (`internal/services/game/`): Canonical rules and campaign state; gRPC APIs under `internal/services/game/api/grpc/`; owns the game database.
- **MCP service** (`internal/services/mcp/`): JSON-RPC adapter for the MCP protocol; forwards to the game service and does not own rules or state.
- **Admin service** (`internal/services/admin/`): HTTP admin dashboard; renders UI and calls the game service for data.
- **Auth service** (`internal/services/auth/`): Authentication domain logic and gRPC API surface; owns the auth database.
- **Connections service** (`internal/services/connections/`): Directed user contact APIs and connection metadata; owns the connections database.
- **AI service** (`internal/services/ai/`): AI credential and agent domain logic + gRPC API surface; owns the AI database.

Non-service utilities live in shared layers:

- **RNG/seed generation**: `internal/services/game/core/random/` (shared domain utility, not a service).
- **Request-scoped identity context helpers**: `internal/platform/requestctx/` (transport-agnostic context primitives reused across services).
- **Cross-service auth introspection client**: `internal/services/shared/authctx/` (shared contract client for auth HTTP token introspection).
- **Seeding CLI**: `cmd/seed` (dev tooling that calls the game service APIs).
