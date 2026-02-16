---
title: "Event Catalog Generator"
parent: "Events"
nav_order: 2
---

# Event Catalog Generator

## Regenerating the catalog
Run the generator from the repo root:

```bash
go generate ./internal/services/game/domain/campaign/event
```

This writes the [event catalog](event-catalog.md) using the Go source of core and Daggerheart events.

Note: runtime event append/validation uses `internal/services/game/domain/event`.
The catalog generator currently sources core event type declarations from
`internal/services/game/domain/campaign/event` plus system event-type
declarations.

## CI check
The Go tests workflow regenerates the catalog and fails if the [event catalog](event-catalog.md) is out of date.
