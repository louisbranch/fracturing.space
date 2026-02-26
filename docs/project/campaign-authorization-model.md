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

Current enforcement note:

- Character mutation authorization is driven by campaign access + ownership
  invariants, not by gameplay role labels. This keeps `GM`/`PLAYER` orthogonal
  to campaign governance while still allowing participants to mutate characters
  they own.

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
| View campaign resources (campaign/participant/character/session reads) | Allow | Allow | Allow | Allow |
| Fork campaign (`ForkCampaign`) | Allow | Allow | Allow | Deny |
| View campaign lineage (`GetLineage`) | Allow | Allow | Allow | Allow |
| View invite resources (invite reads) | Allow | Allow | Allow | Deny |
| Update campaign metadata/settings | Allow | Allow | Allow | Deny |
| End/archive/restore campaign | Allow | Allow | Allow | Deny |
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

Canonical implementation table:

- Runtime role/action/resource policy lives in
  `internal/services/game/domain/authz/policy.go`.
- Transport boundaries call the canonical evaluator instead of re-defining role
  matrices in handler code.
- Campaign-scoped coverage now includes event feeds
  (`ListEvents`/`ListTimelineEntries`/`SubscribeCampaignUpdates`), snapshot
  reads/writes (`GetSnapshot`/`PatchCharacterState`/`UpdateSnapshotState`), and
  fork APIs (`ForkCampaign`/`GetLineage`). `ForkCampaign` is a campaign
  governance action and enforces `CapabilityManageCampaign`; `GetLineage`
  remains read-level.

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

- Ownership is persisted in character projection state as
  `owner_participant_id`.
- Projection owner state is sourced from `character.created`
  (`owner_participant_id`, with actor-id fallback for created events) and from
  `character.updated.fields.owner_participant_id` transfers.
- Runtime authorization reads owner from projection-backed character state
  (request-path auth checks do not replay event history).
- Ownership transfer is represented by `character.updated` with
  `fields.owner_participant_id` and is restricted to campaign `OWNER` access in
  server authorization.
- Controller assignment currently uses `participant_id` in
  `character.updated` payloads (`SetDefaultControl` path).

## AuthorizationService.Can Target Semantics

`AuthorizationService.Can` accepts optional `AuthorizationTarget` context.

Participant governance behavior (action/resource = `manage` + `participant`):

1. `target_participant_id` (or fallback `resource_id`) can identify the target.
2. `target_campaign_access` may be provided directly; when omitted and target id
   is present, server policy attempts to resolve current target access.
3. `requested_campaign_access` triggers access-change invariant checks using the
   same domain evaluator as participant write paths.
4. Managers are denied when mutating owner targets
   (`AUTHZ_DENY_TARGET_IS_OWNER`) and when assigning owner access
   (`AUTHZ_DENY_MANAGER_OWNER_MUTATION_FORBIDDEN`).
5. Final-owner demotion is denied with
   `AUTHZ_DENY_LAST_OWNER_GUARD` when applicable.
6. `participant_operation` target context can be used to disambiguate
   participant checks:
   - `MUTATE`: baseline participant mutation checks.
   - `ACCESS_CHANGE`: access-change invariant checks (requires
     `requested_campaign_access`).
   - `REMOVE`: participant-removal checks including final-owner and
     active-owned-character guards.

Character mutation behavior (action/resource = `mutate` + `character`) evaluates
ownership (`owner_participant_id` from target, or projection owner resolved from
`resource_id`) for member-level mutation guards.

## Fork Policy

- `ForkCampaign` requires campaign governance (`OWNER`/`MANAGER`/`ADMIN`)
  because it creates a new campaign branch.
- `GetLineage` remains campaign-read scoped and is available to campaign
  members.
- Future TODO: starter campaigns may adopt an open-fork policy that allows
  non-members to fork into a new campaign where they become owner.

## Batch Authorization Guidance

- Web surfaces should prefer batched authorization checks for show/hide and
  per-row actions instead of issuing many unary `Can` requests.
- `AuthorizationService.BatchCan` accepts repeated checks (`check_id`,
  `campaign_id`, `action`, `resource`, optional `target`) and returns per-check
  results with echoed `check_id`.
- Batch results are returned in request order and should be correlated by
  `check_id` on the client.
- Batch item evaluation uses the same policy path as unary `Can` (same reason
  codes and actor resolution behavior).
- Invalid batch payloads fail the whole request (fail-fast), matching unary
  input validation semantics.
- Batch checks should carry full target context (resource id, target
  participant access, requested access) so server policy decisions match write
  invariants.

## Web Mutation Gate Guidance

- Campaign mutation routes in web must require an evaluated authorization
  decision from the game authorization service.
- If authz is unavailable or decision evaluation is missing, web must deny
  mutation actions (fail closed).

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
- `AUTHZ_DENY_MANAGER_OWNER_MUTATION_FORBIDDEN`
- `AUTHZ_DENY_NOT_RESOURCE_OWNER`
- `AUTHZ_DENY_TARGET_OWNS_ACTIVE_CHARACTERS`
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
