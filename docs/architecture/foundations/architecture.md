---
title: "System architecture"
parent: "Foundations"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-09"
---

# System Architecture

Concise architecture map for service boundaries, domain ownership, and storage
shape.

## Layered model

Fracturing.Space is organized into four layers:

- **Transport**: service entrypoints (`cmd/game`, `cmd/auth`, `cmd/social`, `cmd/discovery`, `cmd/ai`, `cmd/notifications`, `cmd/status`, `cmd/userhub`, `cmd/worker`, `cmd/mcp`, `cmd/chat`, `cmd/admin`, `cmd/web`).
- **Platform**: shared infrastructure (`internal/platform/`).
- **Domain**: game/auth/social domain logic under `internal/services/*/domain`.
- **Storage**: service-owned SQLite adapters and data files.

The MCP service is an adapter surface; rules/state authority remains in game and
auth/social domain services.

## Core service boundaries

- **Game** (`internal/services/game/`): canonical campaign/session/action rules and event-sourced state.
- **Web** (`internal/services/web/`): browser-facing modular BFF.
- **Admin** (`internal/services/admin/`): operator-facing web surface.
- **MCP** (`internal/services/mcp/`): JSON-RPC tool/resource bridge.
- **Auth** (`internal/services/auth/`): identity/authentication and OAuth primitives.
- **Social** (`internal/services/social/`): profile metadata, contacts, and authenticated people-search read models built from auth-owned identity data.
- **Discovery** (`internal/services/discovery/`): public discovery entry metadata and future public browsing indexes, not authenticated invite search.
- **AI** (`internal/services/ai/`): AI credential/agent orchestration.
- **Notifications** (`internal/services/notifications/`): user inbox intent and channel-delivery orchestration.
- **Status** (`internal/services/status/`): capability health and override state authority.
- **Userhub** (`internal/services/userhub/`): experience read-model aggregation.
- **Worker** (`internal/services/worker/`): asynchronous outbox and scheduled processing runtime, including worker-owned delivery of game invite notification intents.
- **Chat** (`internal/services/chat/`): optional session-scoped realtime transcript delivery for human participants. It is not the authority for gameplay routing, AI pacing, or rules-affecting communication state.

Each service owns transport, orchestration, domain logic, and storage adapters
within its boundary.

Interaction and transcript boundary:

- `game` owns authoritative active-play state through `game.v1.InteractionService`: active scene, scene player phases, OOC pause/resume, and AI turn pacing.
- `chat` owns optional human-only session transcript transport: websocket fanout, per-connection subscriptions, sequencing, and history.
- `chat` validates campaign/session membership through game/auth, but it does not consume or broadcast gameplay workflow state.
- `web` must render active play from game-owned interaction state and may use chat only as a separate optional transcript surface.

Authenticated surface: canonical `/app/*` routes (`/app/dashboard`, `/app/campaigns`, `/app/campaigns/{id}/*`, `/app/notifications`, `/app/settings/*`).

## Game domain architecture

Game service combines core domain packages and system extension packages.

- Core domains: `campaign`, `participant`, `character`, `invite`, `session`, `action`, etc.
- System extensions: `internal/services/game/domain/bridge/<system>/`.
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
