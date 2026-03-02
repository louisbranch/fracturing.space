---
title: "Events"
nav_order: 6
has_children: true
---

# Events

Generated command/event contract artifacts for the event system.

## Read order

1. [Event catalog](event-catalog.md) for event types and payload schemas.
2. [Command catalog](command-catalog.md) for command definitions and validators.
3. [Event usage map](usage-map.md) for emitter/applier wiring.

## Regeneration

Run from repo root:

```bash
go run ./internal/tools/eventdocgen
```

This updates:

- `docs/events/event-catalog.md`
- `docs/events/command-catalog.md`
- `docs/events/usage-map.md`

## Related architecture

- [Event-driven system](../architecture/foundations/event-driven-system.md)
- [Game systems architecture](../architecture/systems/game-systems.md)
