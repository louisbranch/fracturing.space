---
title: "Campaign Authorization Model"
parent: "Project"
nav_order: 15
---

# Campaign Authorization Model

## Purpose

Define the canonical authorization model for campaign operations and gameplay
control.

This document is a clean-slate policy contract. Backward compatibility with
older authorization behavior is not a goal.

## Scope

This model covers:

- Platform-level administration.
- Campaign-level governance.
- Gameplay role authority.
- Participant-owned resource authority (starting with characters).

## Policy Axes

### Platform Role

- `ADMIN`: global operator role with override capability.

### Campaign Access

- `OWNER`: highest campaign governance authority.
- `MANAGER`: delegated campaign governance authority.
- `MEMBER`: standard participant authority.

### Gameplay Role

- `GM`
- `PLAYER`

Gameplay role is independent from campaign governance access. A `GM` is not
implicitly a campaign `OWNER` or `MANAGER`.

### Resource Relationship

- `RESOURCE_OWNER`: participant owns the resource.
- `RESOURCE_CONTROLLER`: participant controls operational use of resource.
- `SELF`: actor is the same participant as target.

## Core Rules

1. Server-side authorization is authoritative for all write actions.
2. Deny by default when no explicit allow rule exists.
3. Campaign governance uses `OWNER`/`MANAGER`/`MEMBER` only.
4. Gameplay role (`GM`/`PLAYER`) governs gameplay actions, not governance actions.
5. `ADMIN` overrides require explicit reason and telemetry/audit record.

## Permission Matrix

| Action | ADMIN | OWNER | MANAGER | MEMBER |
|---|---|---|---|---|
| View campaign resources (participant/character/session/invite reads) | Allow | Allow | Allow | Allow |
| Update campaign metadata/settings | Allow | Allow | Deny | Deny |
| End/archive/restore campaign | Allow | Allow | Deny | Deny |
| Transfer campaign ownership | Allow | Allow | Deny | Deny |
| Promote/demote participants across access levels | Allow | Allow | Limited | Deny |
| Create participant | Allow | Allow | Allow | Deny |
| Update participant role/controller | Allow | Allow | Limited | Deny |
| Remove participant from campaign | Allow | Allow | Limited | Deny |
| Create/revoke invites | Allow | Allow | Allow | Deny |
| Create character | Allow | Allow | Allow | Allow (owned only) |
| Update character metadata | Allow | Allow | Allow | Allow (owned only) |
| Delete character | Allow | Allow | Allow | Allow (owned only) |
| Assign character controller | Allow | Allow | Allow | Deny |
| Transfer character ownership | Allow | Allow | Deny | Deny |
| Start/end session | Allow | Allow | Allow | Deny |
| Session gate management (open/resolve/abandon) | Allow | Allow | Allow | Deny |
| Gameplay GM-only actions | Allow | Allow if GM | Allow if GM | Allow if GM |

## Matrix Notes

`Limited` means:

- `MANAGER` cannot change `OWNER` access.
- `MANAGER` cannot assign or remove `OWNER`.
- `MANAGER` can manage `MEMBER` records and operational fields for `MANAGER`
  peers when explicitly permitted by policy implementation.

`Allow (owned only)` means:

- Member action is allowed only when actor is `RESOURCE_OWNER`.

Phase note:

- Phase 2 enforcement requires campaign membership for character create/update/delete/profile writes and manager-or-owner access for controller assignment.
- Phase 3 ownership enforcement now:
  - Enforces `RESOURCE_OWNER` checks for member update/delete/profile-write actions.
  - Stamps `owner_participant_id` on `character.created` payloads.
  - Supports explicit ownership transfer through
    `UpdateCharacter.owner_participant_id` (owner-only governance action).
  - Blocks participant removal only when the participant is the current owner of
    at least one active character.

## Invariants

1. A campaign must always have at least one `OWNER`.
2. Self-escalation is forbidden (`MEMBER` cannot make self `MANAGER`).
3. `MANAGER` cannot elevate any participant to `OWNER`.
4. `OWNER` cannot remove/demote the final remaining `OWNER`.
5. Ownership transfer must be explicit, atomic, and audited.
6. Ownership and control are separate fields and must not be conflated.
7. Participant removal requires no active owned resources unless ownership is
   reassigned first.

## Character Ownership Contract

Characters use two distinct relationships:

- `owner_participant_id`: governance owner for edit/delete/transfer authority.
- `controller_participant_id`: operational controller for gameplay flow.

Rules:

1. Owner defaults to the creator participant when created through participant
   context.
2. Members may mutate only characters they own unless elevated by access level.
3. Controller assignment does not transfer ownership.
4. Ownership transfer is restricted to `OWNER` or `ADMIN`.
5. Participant leave/removal is blocked while the participant owns active
   characters.

Implementation note:

- Ownership is resolved from both `character.created` payload
  (`owner_participant_id`, with actor-id fallback for legacy events) and
  `character.updated` payload field `owner_participant_id`.
- Ownership transfer is represented by `character.updated` with
  `fields.owner_participant_id` and is restricted to campaign `OWNER` access in
  server authorization.
- Controller assignment currently uses `participant_id` in
  `character.updated` payloads (`SetDefaultControl` path).

## Override Model

`ADMIN` overrides are allowed for operational recovery and moderation:

1. Override requires a non-empty reason.
2. Override must record actor, target, action, and reason in telemetry.
3. Domain mutation still emits normal domain events.
4. Denied override attempts are telemetry events, not domain events.
5. Runtime override signal (game gRPC write auth boundary):
   - `x-fracturing-space-platform-role: ADMIN`
   - `x-fracturing-space-authz-override-reason: <non-empty reason>`

Runtime semantics:

- Missing/empty override reason is denied with
  `AUTHZ_DENY_OVERRIDE_REASON_REQUIRED`.
- Successful override emits telemetry with `decision=override` and
  `reason_code=AUTHZ_ALLOW_ADMIN_OVERRIDE`.

`OWNER` may override `MANAGER` and `MEMBER` actions within campaign scope, but
cannot violate hard invariants (for example, final-owner removal).

## Audit and Telemetry Expectations

Successful mutations:

- Recorded in the campaign event journal as domain events.

Authorization decisions:

- Recorded as telemetry for allow/deny/override outcomes.
- Must include policy reason code, actor identity, target identity, request ID,
  and invocation ID.

## Reason Codes (Policy Surface)

Implementations should emit stable machine-readable policy reason codes, for
example:

- `AUTHZ_DENY_ROLE_REQUIRED`
- `AUTHZ_DENY_ACCESS_LEVEL_REQUIRED`
- `AUTHZ_DENY_TARGET_IS_OWNER`
- `AUTHZ_DENY_LAST_OWNER_GUARD`
- `AUTHZ_DENY_NOT_RESOURCE_OWNER`
- `AUTHZ_ALLOW_ADMIN_OVERRIDE`

## Phased Rollout

Phase 1: Documentation and policy ratification

- Publish this model and finalize matrix semantics.

Phase 2: Central server evaluator

- Implement one game-service authorization evaluator reused by all write paths.

Phase 3: Ownership semantics

- Add explicit ownership vs control enforcement for participant-owned resources.

Phase 4: Full auth telemetry

- Emit and monitor allow/deny/override decision telemetry across all write APIs.

Phase 5: Hardening

- Remove redundant client-side assumptions once server enforcement is complete.
