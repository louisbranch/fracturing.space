---
title: "Domain language"
parent: "Foundations"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-08"
---

# Domain Language

Canonical terminology for packages, APIs, and docs.

## Identity and account terms

- **User**: core identity record keyed by user ID.
- **Username**: immutable auth-owned account locator and public handle.
- **Passkey**: primary authentication credential (multiple per user allowed).
- **Recovery code**: single-use offline emergency credential returned once at
  signup completion and rotated after successful recovery.
- **Recovery session**: narrow auth-owned state that permits only replacement
  passkey registration after recovery-code verification.
- **Web session**: durable authenticated browser session created only after
  successful passkey login or successful recovery completion.
- **Contact**: owner-scoped social/discovery quick-access relationship.

Identity/auth boundaries: [Identity and OAuth](../platform/identity-and-oauth.md).

## Campaign model terms

- **Campaign**: complete event history + derived views for a game timeline.
- **Session**: active gameplay view within a campaign.
- **Participant**: campaign seat with governance/gameplay role context.
- **Character**: participant-owned or controlled gameplay identity.
- **Fork**: new campaign timeline branched from another campaign sequence.

Campaign metadata concerns:

- **Status**: gameplay lifecycle (`draft`, `active`, `completed`, `archived`).
- **Intent**: campaign purpose (`standard`, `starter`, `sandbox`).
- **Access policy**: discoverability (`private`, `restricted`, `public`).

## Event-sourcing terms

- **Event journal**: immutable append-only campaign log.
- **Event authority**: accepted mutations are represented by events in journal.
- **Event**: immutable fact accepted by domain rules.
- **Projection**: derived query state built from events.
- **Snapshot**: materialized derived state at a sequence boundary.
- **Checkpoint**: replay cursor tracking last applied sequence.

Write-path semantics: [Event-driven system](event-driven-system.md).

## Story terminology

- **Story content**: reusable campaign-agnostic narrative material.
- **Story events**: campaign-specific narrative facts stored in event journal.

Examples: note additions, canonized facts, scene start/end, reveals.

## Session governance terms

- **Session gate**: temporary checkpoint blocking action flow until resolved.
- **Session start readiness**: invariant evaluated before `session.start` is
  accepted.
- **Session readiness blocker**: one unmet readiness invariant surfaced with a
  stable machine-readable code, operator-readable message, and optional
  metadata.

Readiness is true when core participant/controller invariants and
system-specific readiness invariants are both satisfied. Reports may include
boundary blockers (campaign status, active session) in addition to core
participant/character blockers.

## Operational read-model terms

- **Join grant JTI index**: projection used to enforce single-use join claims.
- **Telemetry**: non-mutating operational/audit records stored outside canonical
  domain event stream.

## Naming principles

1. Prefer domain terms over storage/transport terms.
2. Keep event type names readable and ownership-scoped.
3. Avoid naming that hard-codes implementation details.

## Error architecture

Domain errors use `apperrors.Error` to carry a stable rejection code, structured
metadata, and an i18n key. Rejection codes are stable machine-readable strings
intended for integrations and client branching; messages are diagnostic and not
guaranteed stable. Transport layers convert domain codes to gRPC status codes via
`HandleError()` (in `grpcerror`), keeping the domain layer transport-agnostic.
`CodeUnknown` is reserved for unexpected/internal errors that do not map to a
specific domain condition — all other rejection reasons use explicit domain codes.

## Non-goals

- Preserving legacy event API vocabulary by default.
- Freezing namespaces before domain language stabilizes.
