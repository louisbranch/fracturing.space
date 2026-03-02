---
title: "System architecture"
parent: "Foundations"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# System Architecture

Concise architecture map for service boundaries, domain ownership, and storage
shape.

## Layered model

Fracturing.Space is organized into four layers:

- **Transport**: service entrypoints (`cmd/game`, `cmd/auth`, `cmd/social`, `cmd/ai`, `cmd/userhub`, `cmd/mcp`, `cmd/admin`).
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
- **Social** (`internal/services/social/`): discovery/profile metadata and contacts.
- **Listing** (`internal/services/listing/`): public listing metadata.
- **AI** (`internal/services/ai/`): AI credential/agent orchestration.
- **Userhub** (`internal/services/userhub/`): experience read-model aggregation.

Each service owns transport, orchestration, domain logic, and storage adapters
within its boundary.

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
