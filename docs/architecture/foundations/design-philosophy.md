---
title: "Design philosophy"
parent: "Foundations"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-11"
---

# Design Philosophy

Classification guide for placing new code within the game service package
hierarchy. Use these decision criteria when adding functionality — the
boundaries exist to keep the codebase navigable and the dependency graph
acyclic as the number of game systems grows.

## Package Tiers

The game service has four tiers, ordered by allowed dependency direction:

```
platform  ←  core  ←  domain  ←  domain/systems
```

Each tier may import from tiers to its left but never to its right.

### `internal/platform/`

Shared infrastructure used by all services: ID generation, error types,
request context, i18n, storage primitives, and observability plumbing.

**Put code here when:** the concept is service-agnostic (not game-specific)
and multiple services could use it. Platform code never references game
domain types.

### `internal/services/game/core/`

System-agnostic RPG mechanics primitives: dice rolling, difficulty checks,
cryptographic random seeding, and encoding helpers.

**Put code here when:**

- The concept applies to any tabletop RPG system, not just Daggerheart.
- The function is a pure mechanical primitive (stateless, deterministic
  given inputs, no domain-model references).
- It needs to be injectable for deterministic tests (e.g. seeded random).

**Do not put code here when:**

- The concept references campaign, session, participant, or event types
  (use `domain/` instead).
- The concept is specific to one game system (use `domain/systems/<system>/`).

**Examples:** `dice.Roll(n, sides, seed)`, `check.Against(dc, modifier)`,
`random.NewSeed()`.

### `internal/services/game/domain/`

Core game domain: aggregates, events, commands, decisions, and engine
orchestration. This tier owns campaign/session/participant/character
lifecycle, the event schema, and replay/fold logic.

**Put code here when:**

- The concept is owned by the campaign/session/action model regardless
  of which game system is active.
- The type is an aggregate state, event payload, command definition,
  decision function, or engine component.
- The behavior is deterministic and replay-safe (no I/O, no wall-clock
  time, no context dependencies).

**Do not put code here when:**

- The concept implements rules from a specific game system — use
  `domain/systems/<system>/` and register it through the module interface.
- The concept is a read-model or storage contract — use `storage/` or
  `projection/`.

**Subpackage conventions:**

| Package | Contains |
|---------|----------|
| `domain/aggregate/` | Aggregate state, fold registry, state helpers |
| `domain/command/` | Command registry and decision flow |
| `domain/engine/` | Handler orchestration (validate → gate → load → decide → persist) |
| `domain/event/` | Event type registry, intent classification, addressing |
| `domain/module/` | Module interface for game system extension |
| `domain/<entity>/` | Per-entity state, events, folds, decisions |

### `domain/systems/` and `domain/systems/<system>/`

System-specific mechanics: adapters, state factories, deciders, folds,
and projection hooks for a particular game system (e.g. Daggerheart).

**Put code here when:**

- The logic implements rules from a specific tabletop RPG system.
- The type is a system-specific event payload, state extension, or
  decision function.
- The adapter applies system events to system-specific projection stores.

**Entry points:**

| Package | Role |
|---------|------|
| `bridge/manifest/` | System descriptor registry — declares modules, adapters, and projection stores |
| `bridge/daggerheart/` | Reference implementation for the Daggerheart system |

See the [game systems guide](../systems/game-systems.md) for the full
contributor workflow.

## Decision Checklist

When adding a new type or function, walk through these questions:

1. **Is it service-agnostic infrastructure?** → `platform/`
2. **Is it a pure mechanical primitive with no domain model references?** → `core/`
3. **Does it reference campaign/session/event types but not a specific game system?** → `domain/`
4. **Does it implement rules or state for a specific game system?** → `domain/systems/<system>/`

If the answer is ambiguous, prefer the **most specific tier** — code is
easier to generalize later than to disentangle.

## Dependency Rules

- `core/` imports only `platform/errors` (no other upward imports).
- `domain/` imports `core/` and `platform/` but never `bridge/` or
  `storage/`.
- `domain/systems/` imports `domain/` and `core/` but never `storage/`
  directly — descriptor-owned adapter builders may extract system stores from
  concrete backend sources, but that extraction logic stays inside the system
  descriptor rather than a shared manifest bundle.
- `storage/` implements domain contracts; it never defines domain behavior.
- `projection/` applies events to stores; it imports domain types and
  storage contracts but never transport types.
- Transport layers (gRPC, MCP, web) import domain, storage, and
  projection but never each other.
