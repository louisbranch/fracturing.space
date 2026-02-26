---
title: "Daggerheart Event Timeline Contract"
parent: "Project"
nav_order: 23
---

# Daggerheart Event Timeline Contract

This document maps high-traffic Daggerheart mechanics onto the canonical write path:

`request -> command -> decider -> event append -> projection apply policy`

Use this as the onboarding contract for new mechanics and for review of existing paths.

This document is intentional/mechanic mapping guidance, not a generated type
inventory. For exact payload fields and emitter references, use:

- [Event catalog](../events/event-catalog.md)
- [Command catalog](../events/command-catalog.md)
- [Usage map](../events/usage-map.md)

## Command/Event Timeline Map

| Mechanic | Command Type(s) | Emitted Event Type(s) | Projection Targets | Apply policy notes | Required invariants |
| --- | --- | --- | --- | --- | --- |
| Action roll resolution | `action.roll.resolve` | `action.roll_resolved` | Event journal (no direct Daggerheart projection mutation) | Request path records event; projection apply is skipped for this envelope | Campaign/session valid; roll payload valid; command must emit event |
| Roll outcome finalization | `action.outcome.apply` | `action.outcome_applied` | Event journal (plus follow-on Daggerheart/system commands) | Request path records outcome event; projection apply is skipped for this envelope | Roll event exists and matches session; no duplicate/bypass apply |
| Outcome-driven GM Fear update | `sys.daggerheart.gm_fear.set` | `sys.daggerheart.gm_fear_changed` | Daggerheart snapshot (`gm_fear`) | Inline apply depends on runtime mode; outbox mode must not inline apply in request path | Fear bounds and spend/gain checks enforced |
| Outcome-driven character state patch | `sys.daggerheart.character_state.patch` | `sys.daggerheart.character_state_patched` | Daggerheart character state | Inline apply mode-controlled | Patch payload must include meaningful deltas |
| Outcome-driven condition change | `sys.daggerheart.condition.change` | `sys.daggerheart.condition_changed` | Daggerheart character conditions | Inline apply mode-controlled | Normalized set diff; no empty/invalid conditions |
| Session gate for GM consequence | `session.gate_open`, `session.spotlight_set` | `session.gate_opened`, `session.spotlight_set` | Session gate + spotlight projections | Inline apply mode-controlled | One open gate at a time; request/session correlation |
| Character damage apply | `sys.daggerheart.damage.apply` | `sys.daggerheart.damage_applied` | Daggerheart character HP/armor | Inline apply mode-controlled | Campaign system is Daggerheart; damage payload valid; emits event |
| Multi-target damage apply | `sys.daggerheart.multi_target_damage.apply` | N × `sys.daggerheart.damage_applied` | Per-target Daggerheart character HP/armor | Inline apply mode-controlled | All targets validated atomically; emits N damage_applied events in single batch via DecideFuncMulti |
| Adversary damage apply | `sys.daggerheart.adversary_damage.apply` | `sys.daggerheart.adversary_damage_applied` | Daggerheart adversary HP/armor | Inline apply mode-controlled | Adversary exists in session; payload valid; emits event |
| Rest | `sys.daggerheart.rest.take` | `sys.daggerheart.rest_taken`, optional `sys.daggerheart.countdown_updated` (when `long_term_countdown` is present) | Daggerheart snapshot and targeted character state, plus long-term countdown state | Inline apply mode-controlled | Rest type valid; campaign/session mutate gates pass; rest + optional countdown update emit atomically from one command decision |
| Downtime move | `sys.daggerheart.downtime_move.apply` | `sys.daggerheart.downtime_move_applied` | Daggerheart character state | Inline apply mode-controlled | Move is valid; resulting resource bounds valid |
| Temporary armor apply | `sys.daggerheart.character_temporary_armor.apply` | `sys.daggerheart.character_temporary_armor_applied` | Daggerheart temporary armor buckets and armor totals | Inline apply mode-controlled | Source/duration/amount validation; emits event |
| Loadout swap and associated resource mutation | `sys.daggerheart.loadout.swap`, `sys.daggerheart.stress.spend` | `sys.daggerheart.loadout_swapped`, `sys.daggerheart.character_state_patched` | Daggerheart character loadout-facing stress/state | Inline apply mode-controlled for Daggerheart events | Recall cost bounds; stress spend consistency |
| Character conditions apply endpoint | `sys.daggerheart.condition.change`, `sys.daggerheart.character_state.patch` (life state updates) | `sys.daggerheart.condition_changed`, `sys.daggerheart.character_state_patched` | Character conditions/life state | Inline apply mode-controlled | No-op updates rejected; roll correlation checked when provided |
| Adversary condition changes | `sys.daggerheart.adversary_condition.change` | `sys.daggerheart.adversary_condition_changed` | Adversary conditions | Inline apply mode-controlled | No-op updates rejected; normalized set required |
| Countdown create/update/delete | `sys.daggerheart.countdown.create`, `sys.daggerheart.countdown.update`, `sys.daggerheart.countdown.delete` | `sys.daggerheart.countdown_created`, `sys.daggerheart.countdown_updated`, `sys.daggerheart.countdown_deleted` | Daggerheart countdown projections | Inline apply mode-controlled | Countdown bounds/rules validated before command |
| Adversary create/update/delete | `sys.daggerheart.adversary.create`, `sys.daggerheart.adversary.update`, `sys.daggerheart.adversary.delete` | `sys.daggerheart.adversary_created`, `sys.daggerheart.adversary_updated`, `sys.daggerheart.adversary_deleted` | Daggerheart adversary projections | Inline apply mode-controlled | Session-scoped adversary integrity and payload validation |

## ApplyRollOutcome sequencing contract

`ApplyRollOutcome` must preserve this command order for replay-safe ownership:

1. optional `sys.daggerheart.gm_fear.set`
2. per-target optional `sys.daggerheart.character_state.patch`
3. per-target optional `sys.daggerheart.condition.change`
4. final `action.outcome.apply`

Invariants:

- `action.outcome.apply` is journal-facing and must not include system-owned
  effects in `pre_effects`/`post_effects`.
- Daggerheart state mutation is expressed only through explicit `sys.daggerheart.*`
  commands/events.
- Session-side follow-up effects (for example gate open + spotlight set) remain
  core-owned post-effects on `action.outcome.apply`.

### Known Gap: Consequence Atomicity

`ApplyRollOutcome` applies consequence commands sequentially. If command 3 of
5 fails, commands 1-2 are already persisted. This is acceptable because:

1. Each consequence command independently produces valid state — there is no
   intermediate "half-applied" state that violates domain invariants.
2. Idempotency guards prevent double-application on retry.
3. `action.outcome.apply` at the end serves as a completion marker — its
   absence signals that the consequence set is incomplete, enabling retry.
4. Replay recovers intermediate state deterministically from the event journal.

If true multi-command atomicity is needed in the future, follow the
`rest.take` precedent: a single command whose decider emits multiple events
from one decision, all batch-appended atomically.

## Mechanic backlog

Priority missing mechanics (P1-P44), tier classifications, clarification gates,
and recommended build order live in the
[Daggerheart Mechanic Backlog](daggerheart-mechanic-backlog.md). That document
evolves independently as mechanics are implemented.

## Source Field Convention

`CharacterStatePatchedPayload.Source` is an optional discriminator set by
transform commands that emit `character_state_patched` events. It enables
journal queries to distinguish the origin of a patch without inspecting field
patterns or introducing separate event types.

| Transform command | Source value |
| --- | --- |
| `sys.daggerheart.hope.spend` | `hope.spend` |
| `sys.daggerheart.stress.spend` | `stress.spend` |
| `sys.daggerheart.character_state.patch` (direct) | _(empty — generic GM/system adjustment)_ |

When adding new transforms that emit `character_state_patched`, set `Source`
to the originating command's short name (the suffix after `sys.daggerheart.`).

## Design Principle: Prefer DecideFuncMulti for Multi-Consequence Atomicity

When a single mechanic produces multiple consequences (N damage events, a
rest plus a countdown update, etc.), prefer emitting all events from one
command decision via `DecideFuncMulti` rather than executing sequential
commands.

**Why**: A single command decision → batch append is atomic. Sequential
commands are individually valid but can partially fail — command 3 of 5
succeeds while command 4 fails, leaving the journal in a state that
requires retry logic. The `rest.take` precedent demonstrates the atomic
pattern: one command whose decider emits `rest_taken` plus an optional
`countdown_updated`, all batch-appended in a single call.

**When to use**:

- One mechanic naturally produces N events of the same type (e.g.
  multi-target damage → N × `damage_applied`).
- One mechanic produces events of different types that must succeed or fail
  together (e.g. rest + countdown update).
- The consequence set is known at decision time (not discovered mid-sequence).

**When sequential commands are acceptable**:

- `ApplyRollOutcome` applies consequences that are independently valid —
  each command produces valid state on its own, and `action.outcome.apply`
  at the end serves as a completion marker. See "Known Gap: Consequence
  Atomicity" above for the full rationale.

**Pattern**:

```go
// Atomic: one command, multiple events via DecideFuncMulti
func decideMultiTargetDamage(cmd command.Command, state *SnapshotState, now func() time.Time) command.Decision {
    return module.DecideFuncMulti(cmd, state, func(targets MultiTargetPayload) ([]command.DecisionEvent, *command.Rejection) {
        events := make([]command.DecisionEvent, 0, len(targets.Targets))
        for _, t := range targets.Targets {
            events = append(events, command.DecisionEvent{
                Type: EventTypeDamageApplied,
                // ... per-target payload ...
            })
        }
        return events, nil
    }, now)
}
```

## Non-Negotiable Handler Rules

1. Mutating request handlers must use shared orchestration (`executeAndApplyDomainCommand`).
2. Request handlers must not call direct event append APIs.
3. Request handlers must not call direct projection/storage mutation APIs for domain outcomes.
4. Every mutating command path must reject empty decision events unless explicitly audit-only.
5. Inline projection apply behavior must be controlled only by runtime mode policy.

## Required Guard Tests

Use these tests as baseline architecture guardrails:

- `internal/services/game/api/grpc/systems/daggerheart/write_path_arch_test.go`
- `internal/services/game/api/grpc/systems/daggerheart/domain_write_helper_test.go`
- `internal/services/game/api/grpc/game/domain_write_helper_test.go`

When adding a new mutating mechanic, update/add tests so bypass patterns fail fast.

## Timeline Row Template

Use this template when adding a new mechanic to the timeline table above.
Fill in each column before writing implementation code.

### Implemented mechanics (main table)

```
| <Mechanic name> | `sys.daggerheart.<domain>.<verb>` | `sys.daggerheart.<domain>_<verb_past>` | <projection store(s)> | Inline apply mode-controlled | <domain rules; validation constraints> |
```

### Priority missing mechanics (P-table)

```
| P<N> | <Mechanic gap scenario> | `sys.daggerheart.<domain>.<verb>` | `sys.daggerheart.<domain>_<verb_past>` | <projection store(s)> | Inline apply mode-controlled | <domain rules; validation constraints> |
```

### Column guidance

| Column | What to write |
|--------|---------------|
| Mechanic | Short name describing the player/GM action (e.g., "Character damage apply") |
| Command Type(s) | Dot-separated command types, comma-separated if multiple. Core commands use bare names (`action.roll.resolve`); system commands use `sys.daggerheart.*` namespace |
| Emitted Event Type(s) | Past-tense event types matching the commands. One command may emit N events via `DecideFuncMulti` |
| Projection Targets | Which stores are written: "Daggerheart snapshot", "Character HP/armor", "Event journal (no direct mutation)", etc. |
| Apply Policy Notes | Typically "Inline apply mode-controlled". Use "Journal-only apply path" for audit-only events. Note when core vs system events have different policies |
| Required Invariants | Semicolon-separated domain rules: validation checks, state preconditions, event emission guarantees |

### Naming conventions

- Commands: `sys.daggerheart.<domain>.<verb>` (e.g., `sys.daggerheart.damage.apply`)
- Events: `sys.daggerheart.<domain>_<verb_past>` (e.g., `sys.daggerheart.damage_applied`)
- Core commands omit the `sys.daggerheart.` prefix (e.g., `action.roll.resolve`)

## How To Add A New Daggerheart Mechanic

1. Add command and event registrations in the Daggerheart decider/registry.
2. Add a timeline row in this document before implementation.
3. Implement request handler using shared write orchestration only.
4. Implement/update adapter projection handling for emitted event types.
5. Add Red/Green tests:
   - command/event behavior
   - projection/apply behavior
   - architecture bypass guard where relevant
6. Validate runtime mode behavior (`inline_apply_only`, `outbox_apply_only`, `shadow_only`) is explicit for the new path.
