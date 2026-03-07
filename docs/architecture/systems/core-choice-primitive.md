---
title: "Core Choice Primitive"
parent: "Systems"
nav_order: 5
status: canonical
owner: engineering
last_reviewed: "2026-03-03"
---

# Core Choice Primitive

## Problem Statement

Thirteen Daggerheart scenario files contain unresolved gaps that share a common root cause: the DSL cannot express branching decisions — presenting options, recording a selection, and applying typed consequences.

**Narrative branching (8 scenarios):** A trigger (roll outcome, Fear spend, NPC arrival) should open a set of options. The GM or a character selects one, and the scenario dispatches effects specific to that selection. Examples: spawn-variant selection (rotted/perfected/legion), rumor fanout, mechanical-benefit branching (advantage/bonus-damage/Relentless).

**Stress/consequence branching (5 scenarios):** A roll outcome triggers conditional consequences (stress, damage, Hope loss) that today have no typed expression. Examples: variable stress rolls (1d4), escape-check follow-up chains, state-dependent damage scaling.

All 13 share a structural pattern: **trigger → options → selection → typed consequences**. A single core primitive resolves all of them. See [core-choice-primitive-contracts](../../reference/core-choice-primitive-contracts.md) for full payload contracts, DSL expression, and per-scenario gap resolution.

## Existing Primitives

The platform provides building blocks that compose with — but do not replace — the choice pattern:

- **Session gates** — model "pause and resume" but have no concept of typed option sets, selection, or consequence dispatch.
- **Roll outcome branching** (`action.outcome.apply`) — dispatches unconditional post-effects for a resolved roll; cannot express "present options and branch on selection."
- **Story notes / spotlights** — observational, not decisional; record state but don't model option presentation.

## Design: Core Choice Aggregate

Two commands and two events, following the existing command → decision → event lifecycle.

| Command | Event | Description |
|---------|-------|-------------|
| `choice.present` | `choice.presented` | Opens a choice point with typed options and optional gate |
| `choice.select` | `choice.selected` | Records selection, dispatches consequence effects |

**Key payload elements:** `ChoiceID`, `ChoiceType`, `Options[]` (each with `ID`, `Label`, `Effects[]`), `GatePolicy` (blocking flag). See [contracts doc](../../reference/core-choice-primitive-contracts.md#payload-contracts) for full type definitions.

**Invariants:**
- `Options` must contain ≥ 2 entries with unique IDs.
- `ChoiceID` must be unique within the session.
- `choice.select` must reference an open choice and a valid option ID.
- If blocking, gate opens/resolves atomically with present/select.

**Event intent:** `IntentProjectionAndReplay` — choices appear in session snapshots. The session fold tracks open choices; `choice.presented` adds, `choice.selected` removes.

**Atomic emission:** `choice.select` uses `DecideFuncMulti` to emit `choice.selected` plus system consequence events in a single batch, following the `action.outcome.apply` pattern.

## Composition with Systems

Game systems extend choice consequences via the effect envelope pattern:

1. Selected option's `Effects` list is dispatched.
2. Effects with a `SystemID` route through the system's module registry.
3. System events (`sys.daggerheart.*`) emit as part of the atomic batch.

Core emits `choice.selected`; systems emit their own consequence events. Core never emits system events directly.

## Gate Integration

- **Blocking** (`GatePolicy.Blocking = true`): `choice.present` atomically emits `choice.presented` + `session.gate_opened`; `choice.select` emits `choice.selected` + `session.gate_resolved`. Gate type: `"choice"`.
- **Non-blocking**: No gate interaction; choice can be resolved at any time during play.

## Out of Scope

- Spatial/placement DSL, variable spawning, companion mechanics (19 other scenario gaps).
- Implementation — this document is design-only; command/event registration, fold/adapter wiring, and DSL runtime deferred.

## Open Questions

1. **Choice ownership boundary** — core (`choice.*`) vs system-delegated validation.
2. **Multi-select** — "pick N of M" semantics; add `max_selections` now or later?
3. **Choice timeout** — auto-resolve after N rounds for blocking choices?
4. **Choice visibility** — all participants vs actor-scoped?
5. **Effect ordering** — specify constraints on multi-event consequence ordering?
6. **Nested choices** — re-entrancy rules if a consequence triggers another choice?
