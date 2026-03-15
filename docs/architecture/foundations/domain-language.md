---
title: "Domain language"
parent: "Foundations"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-09"
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
- **Gate response authority**: the domain unit whose response counts toward a
  gate workflow. Current communication gates count participant responses, not
  persona responses.
- **Ready check**: session gate workflow with fixed `ready` / `wait` responses.
- **Vote**: session gate workflow where responses are explicit option choices,
  optionally constrained by workflow metadata.
- **Gate resolution state**: derived workflow summary that indicates whether a
  gate is still collecting responses, is blocked, is ready to resolve, or
  requires manual review.
- **Session start readiness**: invariant evaluated before `session.start` is
  accepted.
- **Session readiness blocker**: one unmet readiness invariant surfaced with a
  stable machine-readable code, operator-readable message, and optional
  metadata.

Readiness is true when core participant/controller invariants and
system-specific readiness invariants are both satisfied. Reports may include
boundary blockers (campaign status, active session) in addition to core
participant/character blockers. The one intentional cross-aggregate
`session.start` workflow is owned by the `domain/readiness` package so those
rules and the first-session activation transition stay in one place.

The one intentional campaign bootstrap workflow,
`campaign.create_with_participants`, is owned by the sibling
`domain/campaignbootstrap` package so the campaign aggregate decider remains
campaign-local even though bootstrap emits participant join events atomically.

## Communication terms

- **Communication stream**: game-owned routed channel for transcript delivery
  and system/control output.
- **Persona**: speaking identity available to one participant in communication
  context. A persona changes who a message is presented as, not which
  participant seat owns rules-affecting decisions.
- **Participant persona**: the participant speaking as themselves.
- **Character persona**: the participant speaking as one controlled character.

Current persona scope is intentionally narrow: participant-self plus eligible
controlled characters. GM/NPC/temporary-mask personas are future extensions,
not current core vocabulary.

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
