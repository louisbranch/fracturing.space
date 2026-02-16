---
title: "Event Catalog Generator"
parent: "Events"
nav_order: 4
---

# Event Catalog Generator

## Regenerating the catalog
Run the generator from the repo root:

```bash
go generate ./internal/services/game/domain/campaign/event
```

This writes:
- the [command catalog](command-catalog.md) using runtime command registry definitions
- the [event catalog](event-catalog.md) using core and Daggerheart event definitions
- the [event usage map](usage-map.md) using emitter/applier scanner output

## Event naming and addressing policy

### Naming

- Use lowercase ASCII event/command types.
- Core command type shape: `<bounded_context>.<entity>.<verb>`.
- Core event type shape: `<bounded_context>.<entity>_<verb_past>`.
- System command/event type shape: `sys.<system_id>.<domain>.<verb_or_verb_past>`.
- System type versioning belongs in `system_version`, not in the type string.
- Daggerheart system types are sys-only (`sys.daggerheart.*`); legacy
  `action.*` system aliases are removed.

### Addressing and identity

- Authoritative ordering identity is `campaign_id + seq`.
- Trace fields: `request_id`, `invocation_id`, `correlation_id`,
  `causation_id`.
- Mutating events require `entity_type` and `entity_id` unless explicitly
  exempted by registry policy.
- System-owned events require `system_id` and `system_version`.

## Source of truth

- Runtime event append/validation uses
  `internal/services/game/domain/event` registries.
- Generated files in this folder are artifacts of the generator and should
  not be edited manually.
- The generator currently sources core event type declarations from
  `internal/services/game/domain/campaign/event` plus system event-type
  declarations.

## CI check
The Go tests workflow regenerates docs and fails if either generated file is out of date:
- [command catalog](command-catalog.md)
- [event catalog](event-catalog.md)
- [event usage map](usage-map.md)
