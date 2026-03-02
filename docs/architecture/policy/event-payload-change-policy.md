---
title: "Event payload change policy"
parent: "Policy and quality"
nav_order: 1
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Event Payload Change Policy

Policy for evolving mutation payloads while preserving replay and projection
correctness.

## Decision summary

1. Delta-only payloads are not the default.
2. Every mutation event defines one authoritative change representation.
3. Authoritative mutation data should be `after`-oriented for deterministic replay.
4. Redundant fields (`before`, `delta`, `added`, `removed`) are informational and
   must pass consistency checks.
5. Full-replace payloads are valid only when replace semantics are explicit.

## Mutation classes

Every mutation event must declare one class.

## `field_patch`

Sparse updates of selected fields.

- authoritative: `*_after` fields or `fields` patch map
- informational: `*_before`, reason/source metadata
- required checks: non-empty patch, unknown-key rejection, optional before/current parity

## `set_replace`

Full logical set/list replacement.

- authoritative: normalized `*_after`
- informational: `added`, `removed`, optional `*_before`
- required checks: normalized `after`; added/removed must match computed diff

## `operation`

Intent-focused operations with resulting state.

- authoritative: operation identity + resulting `after` values when state mutates
- informational: `before`, `delta`, source/roll metadata
- required checks: arithmetic/invariant consistency; explicit no-op policy

## `full_replace`

Explicit full subdocument/record replacement.

- authoritative: full scoped object
- informational: optional reason/source metadata
- required checks: replace scope explicit; projectors assign full scope unconditionally

## Cross-cutting authority rules

1. One event type uses one authoritative representation.
2. Projectors/adapters mutate from authoritative fields only.
3. Informational fields are never required for correctness.
4. Replay compatibility may decode legacy shapes, but deciders emit canonical shape.

## Inventory posture

Current high-mutation domains (`campaign`, `character`, `participant`,
`action`, `sys.daggerheart`) already map to these classes. New event types must
adopt class + authority semantics during review.

Detailed inventories belong in generated catalogs and implementation tickets,
not in this policy page.

## Rollout sequence

1. Classify event type (`field_patch`, `set_replace`, `operation`, `full_replace`).
2. Define authoritative fields in payload docs and validators.
3. Add consistency checks for informational redundancy.
4. Ensure deciders emit canonical shape.
5. Ensure adapters read authoritative fields only.
6. Keep compatibility decoders for legacy historical replay when needed.

## Adoption checklist

- class selected and documented
- authoritative fields identified
- redundancy consistency checks added
- decider emission canonicalized
- adapter consumption uses authoritative fields only
- replay compatibility note captured where legacy payloads exist
