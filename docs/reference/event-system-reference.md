---
title: "Event system reference"
parent: "Reference"
nav_order: 8
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Event System Reference

Detailed reference for event-system mechanics, envelope fields, validation, and
troubleshooting.

For onboarding architecture, start with
[Event-driven system](../architecture/foundations/event-driven-system.md).

## Command envelope fields

Command envelope type: `internal/services/game/domain/command/registry.go`.

Key fields:

- `campaign_id`: required command scope.
- `type`: required command type (`campaign.create`, `participant.update`, etc.).
- `actor_type` and `actor_id`: initiating principal.
- `session_id`: optional session scope.
- `request_id` and `invocation_id`: request tracing through transport boundaries.
- `entity_type` and `entity_id`: target hints for deciders/folders.
- `system_id` and `system_version`: required for system-owned commands.
- `correlation_id` and `causation_id`: optional causal lineage.
- `payload_json`: canonicalized payload.

## Event envelope fields

Event envelope type: `internal/services/game/domain/event/registry.go`.

Key fields:

- `campaign_id`: required scope.
- `type`: event type string (`participant.created`, `sys.daggerheart.*`, etc.).
- `timestamp`: occurrence time.
- `actor_type` and `actor_id`: origin principal.
- `session_id`, `request_id`, `invocation_id`: traceability.
- `entity_type`, `entity_id`: affected entity.
- `system_id`, `system_version`: required for system-owned events.
- `correlation_id`, `causation_id`: lineage.
- `payload_json`: immutable decision payload.
- integrity fields (`seq`, hash/signature metadata): append-time ownership.

## Event intent details

`event.Definition.Intent` controls fold/projection behavior:

- `IntentProjectionAndReplay`: folded + projected.
- `IntentReplayOnly`: folded only.
- `IntentAuditOnly`: journal-only.

Default intent is `IntentProjectionAndReplay` when omitted.

## Decision model

A decision is produced by deciders and consumed by engine write orchestration.

- accepted decisions emit one or more events
- rejected decisions emit no domain mutation event
- mutating command paths should reject empty event decisions unless explicitly
  audit-only by contract

## Registration surfaces

Runtime registration should remain coherent across these surfaces:

- command registry (`domain/command`)
- event registry (`domain/event`)
- module registry (`domain/module`)
- adapter registry (`domain/bridge/adapter_registry.go`)

For system modules, manifest-driven registration in
`internal/services/game/domain/bridge/manifest/manifest.go` is the source of
truth for module + metadata + adapter alignment.

## Validation rules and why they matter

## Fold coverage validation

Replay-relevant events must have fold handlers. Prevents runtime replay holes.

## Adapter coverage validation

Projection-relevant events must have adapter handlers. Prevents silent projection
staleness.

## Audit-only dead handler guard

Audit-only events should not keep fold handlers. Prevents dead/unused fold code.

## Namespace and ownership validation

- core paths must not emit system-owned events
- system paths must stay in `sys.<system_id>.*` namespaces
- system envelopes must carry system identity/version

## Event naming conventions

- Core commands/events use domain nouns (`campaign.*`, `participant.*`, etc.).
- System-owned types use `sys.<system_id>.*` naming.
- Event types should be past-tense facts.
- Command types should be imperative actions.

## Trigger semantics

When modeling behavior:

1. command trigger captures intent
2. decider evaluates invariant checks against state
3. accepted decision emits immutable fact events
4. projection layers consume events based on intent

Trigger evaluation must remain deterministic under replay.

## Projections and consistency reference

- projections are derived and rebuildable
- replay correctness is more important than immediate denormalized convenience
- adapters must be idempotent under repeated event application

For replay and repair operations, use
[Replay operations](../running/replay-operations.md).

## Startup validator troubleshooting

## Missing fold coverage

Symptom: startup/replay failure for a projection/replay event type.

Fix: register a fold handler for every replay-relevant event.

## Missing adapter coverage

Symptom: event appends but projection does not update.

Fix: register adapter handler for every projection-relevant event.

## Intent mismatch

Symptom: replay-only event expected to project, or audit-only event unexpectedly folded.

Fix: align `event.Definition.Intent` with intended behavior and handler coverage.

## Namespace mismatch

Symptom: core emits `sys.*` or system emits core event names.

Fix: keep ownership boundaries explicit in decider output paths.

## Related docs

- [Event-driven system](../architecture/foundations/event-driven-system.md)
- [Event replay architecture](../architecture/foundations/event-replay.md)
- [Game systems architecture](../architecture/systems/game-systems.md)
- [Events generated catalogs](../events/index.md)
