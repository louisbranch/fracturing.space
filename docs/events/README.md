---
title: "Event Catalog Generator"
parent: "Events"
nav_order: 4
---

# Event Catalog Generator

This folder is the contract surface for runtime-registered command/event types.
Treat generated files here as canonical for exact type names, payload schemas,
and emitter/applier references.

## What lives here

- [command-catalog.md](command-catalog.md): command definitions from runtime
  registries.
- [event-catalog.md](event-catalog.md): event definitions and payload mappings
  from runtime registries.
- [usage-map.md](usage-map.md): emitter/applier cross-reference generated from
  source scanning.

## Regenerating catalogs

Run from repo root:

```bash
go run ./internal/tools/eventdocgen
```

This writes:
- the [command catalog](command-catalog.md) using runtime command registry definitions
- the [event catalog](event-catalog.md) using core and Daggerheart event definitions
- the [event usage map](usage-map.md) using emitter/applier scanner output

## Canonical boundaries

- Runtime event append/validation uses
  `internal/services/game/domain/event` registries.
- Runtime command validation/decision uses
  `internal/services/game/domain/command` registries.
- Generated files in this folder are artifacts of the generator and should
  not be edited manually.
- Author conceptual intent and invariants in project docs, not duplicated
  type/payload inventories:
  - [Event-driven system](../project/event-driven-system.md)
  - [Game systems architecture](../project/game-systems.md)
  - [Daggerheart event timeline contract](../project/daggerheart-event-timeline-contract.md)

## CI check

Generated docs are checked by:

- `make event-catalog-check`
- `make integration` (includes `event-catalog-check`)
