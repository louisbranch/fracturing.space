---
title: "Event payload change policy"
parent: "Architecture"
nav_order: 18
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Event Payload Change Policy

This document defines how mutation events must describe record changes so event
streams remain readable, replay-safe, and projection-safe.

## Decision Summary

1. Delta-only payloads are not the default for this codebase.
2. Every mutation event must define one authoritative representation of change.
3. Authoritative state change data should be `after`-oriented for replay and projection determinism.
4. Redundant fields such as `before`, `delta`, `added`, and `removed` are allowed only as informational fields with strict consistency validation.
5. Full-replace payloads are valid only when replace semantics are explicit and projectors apply fields unconditionally.

## Why Delta-Only Is Not the Global Default

1. Pure deltas require current state to interpret, which increases replay fragility when historical events are repaired or reapplied.
2. Projections and adapters can safely rebuild from `after` values without re-deriving intent.
3. Auditing intent still matters; delta and reason fields are retained as informational metadata where useful.

## Canonical Mutation Classes

Every new mutation event must declare one class in design and code review.

### `field_patch`

Use for sparse updates where only selected fields change.

Authoritative fields:
1. `<field>_after` fields, or a `fields` map containing only changed keys and their resulting values.

Informational fields:
1. Matching `<field>_before` values.
2. Optional reason/source metadata.

Validation rules:
1. At least one authoritative changed field must be present.
2. Unknown keys are rejected.
3. If `before` is provided and current snapshot is available, `before` must match current state.

### `set_replace`

Use when the entire logical set/list should be replaced.

Authoritative fields:
1. Normalized `*_after` set/list.

Informational fields:
1. `added` and `removed`.
2. Optional `*_before` set/list.

Validation rules:
1. `*_after` is required and normalized.
2. If `added` or `removed` are present, they must exactly match the diff implied by `before/current -> after`.

### `operation`

Use when event meaning is an operation intent, not only a final field write.

Authoritative fields:
1. Operation identity fields and required resulting value fields (`after` when state outcome is stored).

Informational fields:
1. `before`, `delta`, amount/source, roll metadata.

Validation rules:
1. Operation invariants must hold (for example `amount == before-after` when those fields are present).
2. No-op operations are rejected unless explicitly supported.

### `full_replace`

Use only when replacing an entire subdocument/record is intentional.

Authoritative fields:
1. Entire scoped object payload.

Informational fields:
1. Optional reason/source metadata.

Validation rules:
1. Replace scope must be explicit in event semantics.
2. Projectors/adapters must assign all fields for that scope unconditionally.

## Cross-Cutting Authority Rules

1. One event type, one authoritative representation.
2. Projectors and adapters must mutate from authoritative fields only.
3. Informational fields must never be required for correctness.
4. Replay compatibility may accept legacy payload shapes, but deciders must emit canonical shape.

## Current Inventory and Canonical Targets

This inventory covers current mutation-heavy domains:
`campaign`, `character`, `participant`, `action`, and `sys.daggerheart`.

| Event type(s) | Current shape | Canonical class | Authoritative fields | Migration action |
| --- | --- | --- | --- | --- |
| `campaign.updated`, `character.updated`, `participant.updated` | `fields` map patch | `field_patch` | `fields` | Keep; codify `fields` as authoritative |
| `character.profile_updated` | nested profile object (`system_profile`) | `full_replace` (scoped) | profile object for target system | Keep; document scope as per-system profile replacement |
| `invite.updated` | targeted scalar status update | `field_patch` | `status` | Keep; no change needed |
| `action.outcome_applied` | `applied_changes[]` with `before/after` | `operation` | operation identity + `after` in each change item | Keep; codify `before` as informational audit data |
| `sys.daggerheart.gm_fear_changed` | scalar `before/after` | `field_patch` | `after` | Keep; treat `before` as informational validation aid |
| `sys.daggerheart.character_state_patched` | sparse `*_before/*_after` pairs | `field_patch` | all `*_after` fields present | Keep; enforce `after` authority and `before` consistency when present |
| `sys.daggerheart.condition_changed`, `sys.daggerheart.adversary_condition_changed` | `conditions_after` + optional `added/removed` (+ optional `conditions_before`) | `set_replace` | `conditions_after` | Normalize; keep `added/removed` as informational with strict diff checks |
| `sys.daggerheart.countdown_updated` | `before/after/delta/looped` | `operation` with resulting state | `after` | Keep; validate `before` and `delta` consistency |
| `sys.daggerheart.damage_applied`, `sys.daggerheart.adversary_damage_applied`, `sys.daggerheart.downtime_move_applied`, `sys.daggerheart.rest_taken` | mixed `*_before/*_after` with operation metadata | `operation` + `field_patch` outcome | present `*_after` fields | Keep; require outcome fields when state mutation is implied |
| `sys.daggerheart.adversary_updated` | full nested record payload | `full_replace` | complete adversary payload | Keep as full-replace; require unconditional projector/adaptor assignment |

Create/delete/join/leave-style events remain operation/fact events and are out of
scope for this record-change normalization.

## Consistency Rules for Redundant Fields

1. `before` and `after` both present:
`before` must equal current snapshot value when snapshot is available.
2. Numeric operation fields present:
`delta` and `amount` must match the implied `before/after` arithmetic.
3. Set-replace fields plus operation lists:
`added` and `removed` must equal normalized diff implied by before/current and after.
4. Redundant fields failing consistency checks:
reject in command/event validation rather than relying on projection-time failures.

## Rollout Sequencing

1. Adopt this policy for all new event types immediately.
2. For existing events, classify each mutation event as `field_patch`, `set_replace`, `operation`, or `full_replace` in registry/module docs.
3. Tighten decider and payload validators so authoritative fields and redundancy consistency are enforced pre-append.
4. Update projectors/adapters to read only authoritative fields for correctness.
5. Keep compatibility decoders for legacy events in replay paths where historical payloads exist.
6. Deprecate redundant fields only after consumers are migrated and replay parity is confirmed.

## Acceptance Notes

This policy is accepted when:
1. Every mutation event type has one documented authoritative representation.
2. Every retained redundant field has explicit consistency checks.
3. Migration work items are sequenced and tracked as implementation tickets.
4. Replay and projection behavior remain deterministic during migration.

## Adoption Checklist

Use this checklist when adding or modifying mutation events.

1. Select mutation class: `field_patch`, `set_replace`, `operation`, or `full_replace`.
2. Declare authoritative fields in payload docs and validators.
3. Mark informational fields and encode their consistency checks.
4. Ensure decider emits canonical payload shape.
5. Ensure projector/adapter uses authoritative fields only.
6. Add replay compatibility note if legacy payloads exist.
7. Update event catalogs/docs with the chosen class and authority semantics.
