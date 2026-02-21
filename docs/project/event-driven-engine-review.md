---
title: "Event-Driven Engine Review"
parent: "Project"
nav_order: 11
---

# Event-Driven Engine Review (2026-02-16)

This review focuses on the redesigned game/domain event-driven engine with two goals:

- Preserve event-sourcing invariants under load and failure.
- Make system extension (new system-owned events/mechanics) safer and easier for new developers.

## Severity and Complexity Scale

- Severity:
  - `High`: can break invariants, produce data drift, or create major operational risk.
  - `Medium`: increases bug probability, onboarding friction, or maintenance cost.
  - `Low`: mainly consistency/clarity improvements.
- Complexity:
  - `S`: small focused change.
  - `M`: cross-file change in one layer.
  - `L`: cross-layer refactor and migration.

## Findings

| # | Finding | Severity | Complexity |
| --- | --- | --- | --- |
| 1 | Projection apply is split across inline request handling and optional outbox worker modes, with weak mode boundaries | High | M |
| 2 | Domain handler execution does repeated validation/replay loads, creating avoidable replay cost and confusing state semantics | High | M |
| 3 | Unhandled events can silently no-op in projection adapters/appliers | High | S |
| 4 | System replay/projector contract is under-specified relative to emitted system event volume | High | L |
| 5 | Legacy event model package remains a docs-generation source beside runtime canonical registries | Medium | M |
| 6 | Command execution and projection apply boilerplate is duplicated across many handlers | Medium | M |
| 7 | Event/command naming has a few domain inconsistencies that hurt discoverability | Low | M |
| 8 | Event intent (audit vs state mutation) is implicit, not encoded in contracts | Medium | M |

### 1) Projection apply mode boundaries are not strict

Evidence:

- Inline apply loops are used broadly after `Domain.Execute` in game handlers and system handlers:
  - `internal/services/game/api/grpc/game/campaign_creator.go`
  - `internal/services/game/api/grpc/game/session_application.go`
  - `internal/services/game/api/grpc/game/participant_creator.go`
  - `internal/services/game/api/grpc/systems/daggerheart/actions.go`
- The event store can enqueue outbox rows on append:
  - `internal/services/game/storage/sqlite/store.go`
- Background workers can process outbox rows:
  - `internal/services/game/app/server.go`
  - `docs/project/projection-apply-outbox.md`

Risk:

- Inference: if inline apply and apply-worker mode are both active for the same stream without sequence-idempotent guards, a projection can be applied twice.
- Even when not active together, duplicated pathways make behavior harder for new developers to reason about.

Recommendation:

- Introduce an explicit write mode contract at startup:
  - `inline_apply_only`
  - `outbox_apply_only`
  - `shadow_only`
- Fail fast on invalid flag combinations.
- Make projection apply idempotent by `campaign_id + seq` before enabling outbox apply worker broadly.

### 2) Domain execution does repeated replay/validation work

Evidence:

- `Execute` validates commands and then calls `Handle`, which validates again:
  - `internal/services/game/domain/engine/handler.go`
- `Execute` loads state again after `Handle`:
  - `internal/services/game/domain/engine/handler.go`
- This behavior is currently expected by tests:
  - `internal/services/game/domain/engine/handler_test.go` (`TestExecute_UsesValidatedCommandForAllStateLoads`)

Risk:

- Extra replay cost per command path.
- `Result.State` semantics become harder to trust and easier to misuse.
- More complexity for maintainers trying to optimize or reason about ordering.

Recommendation:

- Refactor handler flow so normalized command and loaded state are computed once.
- Keep one authoritative source for returned state and snapshot persistence.
- Update tests to assert one replay load per execution path unless a gate check explicitly requires extra loading.

### 3) Unhandled events can fail silently

Evidence:

- Core projection applier default path returns `nil` for unknown non-system events:
  - `internal/services/game/projection/applier_domain.go`
- Daggerheart adapter default path also returns `nil` for unknown system events:
  - `internal/services/game/domain/systems/daggerheart/adapter.go`

Risk:

- Adding a new event without wiring projection handling can look successful while read models lag or miss data.

Recommendation:

- Add projection intent metadata to event definitions and enforce it.
- Add parity tests that compare registered event definitions against applier/adapter handling where projection intent is required.

### 4) System projector coverage is under-specified

Evidence:

- Daggerheart decider emits many system events:
  - `internal/services/game/domain/systems/daggerheart/decider.go`
- Daggerheart projector currently only folds `gm_fear_changed`:
  - `internal/services/game/domain/systems/daggerheart/projector.go`

Risk:

- For system commands that should depend on prior system state, replay-derived command-time state may be incomplete.
- New system authors may copy this pattern without realizing the tradeoff.

Recommendation:

- Define a clear contract per system:
  - Which events are command-state-relevant (must be folded by projector).
  - Which events are projection-only or audit-only.
- Encode this in tests and docs for each system module.

### 5) Two event model surfaces remain in active use

Evidence:

- Runtime canonical validation/append uses:
  - `internal/services/game/domain/event/registry.go`
- Docs generation still references legacy package:
  - `docs/events/README.md`
  - `internal/tools/eventdocgen/main.go`
  - `internal/services/game/domain/campaign/event`

Risk:

- New developers can edit or read the wrong package first.
- Terminology and ownership boundaries are harder to learn.

Recommendation:

- Move catalog generation to runtime registry definitions (`domain/event` + registered system modules).
- Remove `internal/services/game/domain/campaign/event` completely once generator and docs are migrated.

### 6) Command execution/apply boilerplate is duplicated

Evidence:

- Many handlers repeat:
  - build `command.Command`
  - call `Domain.Execute`
  - handle rejections
  - loop `Apply` over emitted events
- Examples:
  - `internal/services/game/api/grpc/game/campaign_creator.go`
  - `internal/services/game/api/grpc/game/session_application.go`
  - `internal/services/game/api/grpc/systems/daggerheart/actions.go`

Risk:

- Envelope metadata consistency (`request_id`, `invocation_id`, actor fields, system metadata) can drift.
- Onboarding burden is high because the same failure handling logic is spread everywhere.

Recommendation:

- Add a shared write helper (`executeAndApply`) with strict defaults and typed command builders for core and system commands.

### 7) Naming consistency has a few outliers

Evidence:

- Most commands/events follow domain-scoped patterns, but seat reassignment is flat:
  - `seat.reassign`
  - `seat.reassigned`
  - `internal/services/game/domain/participant/decider.go`

Risk:

- Discoverability and filtering are weaker (`participant.*` queries miss seat transitions).

Recommendation:

- Move to `participant.seat.reassign` and `participant.seat_reassigned`.
- Preserve aliases for migration windows, then deprecate.

### 8) Event intent is implicit

Evidence:

- Event contracts (`event.Definition`) currently encode owner/addressing/validator but not operational intent:
  - `internal/services/game/domain/event/registry.go`

Risk:

- Developers cannot easily tell whether an event is:
  - audit-only
  - expected to mutate projections
  - required for command-time fold/state

Recommendation:

- Add an explicit event intent field in definition metadata and enforce expected handling in tests.

## Quick Wins

1. Add startup guardrails to prevent unsupported projection apply mode combinations.
2. Add parity tests for "event registered but not handled" cases in core applier and system adapters.
3. Add a short "new system event checklist" section to `docs/project/game-systems.md`:
   - command registration
   - event registration
   - projector contract
   - adapter apply contract
   - tests for replay + projection impact
4. Eliminate `campaign/event` as a source in docs generation and remove the package.

## Strategic Improvements

1. Unify write orchestration around one explicit execution/apply coordinator.
2. Encode event intent in registry metadata and enforce it with architecture tests.
3. Consolidate event documentation generation on runtime registries and remove legacy event model package.
4. Define a formal system extension contract:
   - required invariants
   - required tests
   - naming policy
   - failure semantics for unhandled events
5. Complete removal of `internal/services/game/domain/campaign/event` and all build/doc references.

## Recommended Implementation Order

1. High/S-M: mode guardrails + unhandled-event parity tests.
2. High/M: handler replay/validation simplification.
3. Medium/M: shared execute-and-apply helper across handlers.
4. Medium-L: registry intent metadata and complete legacy event package removal.

## Status Update (2026-02-18)

The hardening work completed after this review materially reduced bypass risk in
request handlers for Daggerheart paths:

1. Shared execute-and-apply orchestration is now the standard mutating path.
2. Architecture guard tests enforce common anti-bypass patterns in Daggerheart handlers.
3. Request-path `RollOutcomeStore` shortcut wiring was removed from gRPC store injection.

Remaining review objective still open:

1. Onboarding clarity for new system authors via explicit mechanic-to-event timeline
   documentation and review checklists.

This is addressed by:

- [Game systems](game-systems.md) timeline requirement section.
- [Daggerheart Event Timeline Contract](daggerheart-event-timeline-contract.md).
