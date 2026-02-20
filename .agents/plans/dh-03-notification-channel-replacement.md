# Daggerheart ExecPlan 03: Replace Notification-Like Journal Events with Explicit Notification Path

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

Reference: `PLANS.md`. This document is maintained according to its requirements.

## Purpose / Big Picture

Separate user-facing notifications/activity messages from domain replay journal events, so the journal remains state-authoritative and notifications have an explicit dedicated channel.

## Progress

- [x] (2026-02-16 21:22Z) Created ExecPlan and identified dedicated notification path as target architecture.
- [x] (2026-02-17 21:45Z) Deferred: define notification producer boundary (command handler vs outbox writer). We are not supporting notifications in this phase.
- [x] (2026-02-17 21:45Z) Deferred: define notification schema independent of domain event catalog. We are not supporting notifications in this phase.
- [x] (2026-02-17 21:45Z) Deferred: replace direct reliance on removed non-mutating events. Completed via mutation-first cleanup in `dh-06`; no notification channel was introduced.
- [x] (2026-02-17 21:45Z) Deferred: add tests for notification generation from mutating outcomes. Not applicable while notifications are out of scope.

## Surprises & Discoveries

- Notifications are explicitly out of scope; this plan is kept for record only.

## Decision Log

- Decision: Notifications are not stored as replay-authoritative domain events.
  Rationale: Notification semantics are presentation/workflow concerns, not state mutation facts.
  Date/Author: 2026-02-16 / Codex
- Decision: Skip notification channel implementation until a dedicated consumer/product requirement is added.
  Rationale: There is no supported notification feature in scope for this phase, and adding one now would increase coupling outside the current engine/event-contract cleanup goals.
  Date/Author: 2026-02-17 / Codex

## Outcomes & Retrospective

- Superseded for now: no in-scope notification channel is being added.

## Context and Orientation

Current flows appear to use some domain events as informational markers. This couples UI/UX signaling with replay mechanics and introduces event types that are either ignored or redundant.

## Plan of Work

1. Choose notification write point.
2. Define a minimal notification record shape.
3. Emit notifications from command outcomes and canonical mutation events.
4. Update consumers/tests to use notification channel rather than journal trace events.

## Concrete Steps

1. Add notification interface in app layer (or outbox model) with campaign/session/request correlation.
2. Map key outcomes (roll resolved, resource spend, gm move trigger) to notifications.
3. Remove journal-dependency assumptions from service handlers.
4. Add tests proving notifications are emitted where expected.

## Validation and Acceptance

- No UX notification flow depends on non-mutating journal events.
- Notifications can be generated when canonical mutation events exist.
- Domain replay remains unaffected by notification behavior.

## Idempotence and Recovery

- Notification producers should be idempotent by request/invocation identity.
- Failed notification writes must not rollback domain event append/projection apply.

## Artifacts and Notes

- Notification schema draft to be captured in this plan.
- Candidate transport: dedicated table/outbox or in-memory pub/sub, depending on product needs.

## Interfaces and Dependencies

- Depends on event pruning plan (`dh-02-event-pruning-and-decider-simplification.md`).
- Impacts API handlers and any timeline/activity read paths.
