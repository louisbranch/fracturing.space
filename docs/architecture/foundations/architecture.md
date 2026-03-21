---
title: "System architecture"
parent: "Foundations"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# System Architecture

Concise architecture map for service boundaries, domain ownership, and storage
shape.

## Layered model

Fracturing.Space is organized into four layers:

- **Transport**: service entrypoints (`cmd/game`, `cmd/auth`, `cmd/social`, `cmd/discovery`, `cmd/ai`, `cmd/notifications`, `cmd/status`, `cmd/userhub`, `cmd/worker`, `cmd/mcp`, `cmd/play`, `cmd/admin`, `cmd/web`).
- **Platform**: shared infrastructure (`internal/platform/`).
- **Domain**: game/auth/social domain logic under `internal/services/*/domain`.
- **Storage**: service-owned SQLite adapters and data files.

The MCP service is an internal adapter surface for AI orchestration; rules/state
authority remains in game and auth/social domain services.

## Core service boundaries

- **Game** (`internal/services/game/`): canonical campaign/session/action rules and event-sourced state.
- **Web** (`internal/services/web/`): browser-facing modular BFF.
- **Admin** (`internal/services/admin/`): operator-facing web surface.
- **MCP** (`internal/services/mcp/`): internal AI-to-game JSON-RPC bridge.
- **Auth** (`internal/services/auth/`): identity/authentication and OAuth primitives.
- **Social** (`internal/services/social/`): profile metadata, contacts, and authenticated people-search read models built from auth-owned identity data.
- **Discovery** (`internal/services/discovery/`): public discovery entry metadata and future public browsing indexes, not authenticated invite search.
- **AI** (`internal/services/ai/`): AI credential/agent orchestration.
- **Notifications** (`internal/services/notifications/`): user inbox intent and channel-delivery orchestration.
- **Status** (`internal/services/status/`): capability health and override state authority.
- **Userhub** (`internal/services/userhub/`): experience read-model aggregation.
- **Worker** (`internal/services/worker/`): asynchronous outbox and scheduled processing runtime, including worker-owned delivery of game invite notification intents.
- **Play** (`internal/services/play/`): browser-facing active-play surface, websocket transport, and durable human transcript storage. The legacy standalone `chat` runtime has been retired; transcript transport now lives inside `play`.

Each service owns transport, orchestration, domain logic, and storage adapters
within its boundary.

Interaction and transcript boundary:

- `game` owns authoritative active-play state through `game.v1.InteractionService`: active scene, scene player phases, OOC pause/resume, and AI turn pacing.
- `play` owns browser-facing active-play transport: launch/session exchange, websocket fanout, human transcript storage, reconnect cursors, and typing indicators.
- `play` validates campaign access through auth/game, but it does not become gameplay authority.
- `web` owns the authenticated shell and launches `/app/campaigns/{id}/game` into `play`.

Authenticated surface: canonical `/app/*` routes (`/app/dashboard`, `/app/campaigns`, `/app/campaigns/{id}/*`, `/app/notifications`, `/app/settings/*`).

## Game domain architecture

Game service combines core domain packages and system extension packages.

- Core domains: `campaign`, `participant`, `character`, `invite`, `session`, `action`, etc.
- System extensions: `internal/services/game/domain/systems/<system>/`.
- Registration alignment: module + metadata + adapter descriptors are wired from
  manifest registration.

Campaigns bind to one registered game system (`system_id + system_version`).

## Event-sourced authority

- event journal is authoritative mutation history
- projections/snapshots are derived views
- mutating handlers must emit events through canonical execute-and-apply paths
- direct projection mutation from request handlers is a boundary violation

See [Event-driven system](event-driven-system.md) for write-path contract.

## Storage boundary model

Service boundaries map to service-owned SQLite stores. For game service,
storage is intentionally split:

- events (`game-events`) for append-only truth
- projections (`game-projections`) for mutable derived reads
- content catalog (`game-content`) for imported/static catalog data

No cross-service direct database writes are allowed; cross-boundary behavior
flows through service APIs/contracts.

## Canonical deep docs

- [Domain language](domain-language.md)
- [Event-driven system](event-driven-system.md)
- [Game systems architecture](../systems/game-systems.md)
- [Web architecture](../platform/web-architecture.md)
