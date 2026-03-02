---
title: "Game systems"
parent: "System extension"
nav_order: 1
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Game Systems Architecture

Canonical architecture for extending Fracturing.Space with a game system.

## Reading order

1. [Adding a command/event/system](../../guides/adding-command-event-system.md) (how-to)
2. [Event-driven system](../foundations/event-driven-system.md) (write-path invariants)
3. This page (architecture boundaries and extension surfaces)
4. Daggerheart references:
   - [Daggerheart creation workflow](../../reference/daggerheart-creation-workflow.md)
   - [Daggerheart event timeline contract](../../reference/daggerheart-event-timeline-contract.md)

## Purpose

Core campaign/session infrastructure stays system-agnostic while each ruleset owns
its mechanics.

This separation allows:

- deterministic replay and projection behavior
- independent system evolution by `system_id + system_version`
- clear ownership of command/event definitions

## Ownership boundaries

- **Core-owned commands/events**: campaign/session/participant/invite/character lifecycle.
- **System-owned commands/events**: mechanics specific to a game system.

Non-negotiable invariants:

1. Core must not emit system-owned events.
2. Systems must not emit core-owned events.
3. System-owned envelopes must include `system_id` and `system_version`.
4. System-owned types must use `sys.<system_id>.*` naming.
5. Request handlers must mutate through commands/events only.

## Event intent policy

Every event must declare an intent. Most system events should use projection + replay
intent. Audit-only events must stay journal-only.

Startup validation enforces coverage:

- fold coverage for replay-relevant events
- adapter coverage for projection-relevant events
- no fold handlers for audit-only events

## Extension surfaces

### 1. Domain module registry (write path)

Location: `internal/services/game/domain/module/registry.go`

Responsibilities:

- register system commands and events
- route system commands to deciders
- route system events to folders during replay

### 2. Metadata bridge (transport metadata)

Location: `internal/services/game/domain/bridge/registry_bridge.go`

Responsibilities:

- map `system_id + system_version` to transport-facing metadata
- keep gRPC/MCP system metadata aligned with module registration

### 3. Adapter registry (projection read path)

Location: `internal/services/game/domain/bridge/adapter_registry.go`

Responsibilities:

- route system events to projection adapters
- keep projection updates replay-safe and idempotent

## Single-source registration rule

All three surfaces must be wired from one descriptor in:
`internal/services/game/domain/bridge/manifest/manifest.go`.

If a system is present in one registry and missing in another, registration is
invalid and must be fixed before merge.

## Package layout contract

Reference layout for a system implementation:

- `internal/services/game/domain/bridge/<system>/module.go` (registration)
- `.../decider.go` (command decisions)
- `.../folder.go` (replay fold)
- `.../adapter.go` (projection apply)
- `.../event_types.go` and `.../payload.go` (contracts)

Keep handlers thin and avoid transport logic in domain packages.

## Authoring invariants

- Deciders and folders must be deterministic.
- Adapter `Apply` behavior must be idempotent under replay.
- Event payloads should capture resulting state (absolute values), not deltas.
- Rejection codes should be stable, machine-readable constants.
- Multi-consequence mechanics should prefer single-command atomic emission patterns.

Detailed Daggerheart examples and timeline contracts are maintained in:

- [Daggerheart event timeline contract](../../reference/daggerheart-event-timeline-contract.md)

## Character creation workflow boundary

Character creation workflow APIs are generic at transport level, while step semantics
are owned by each system provider.

Daggerheart workflow and readiness behavior is specified in:

- [Daggerheart creation workflow](../../reference/daggerheart-creation-workflow.md)

## Minimum review checklist

For any new system or mechanic:

1. Command and event registrations are explicit and tested.
2. Replay fold and projection adapter coverage exists for new event types.
3. Mutating paths use shared command execution orchestration.
4. Generated event catalogs are updated (`docs/events/`).
5. At least one happy-path and one rejection-path test exists per new command/event pair.

## Related docs

- [Event-driven system](../foundations/event-driven-system.md)
- [Adding a command/event/system](../../guides/adding-command-event-system.md)
- [Events index](../../events/index.md)
