---
title: "Campaign Authorization Audit and Telemetry"
parent: "Project"
nav_order: 16
---

# Campaign Authorization Audit and Telemetry

## Purpose

Define how authorization outcomes are represented for operational auditing.

Domain events remain the immutable source of truth for successful state
mutations. Authorization decisions that do not produce a domain event (especially
denies) must be captured as telemetry.

## Data Sources

### Domain Event Journal

Use campaign domain events for:

- Successful mutations (`participant.updated`, `character.updated`, etc.).
- Actor/request correlation (`actor_type`, `actor_id`, `request_id`,
  `invocation_id`).

Do not emit domain events for denied authorization attempts.

### Telemetry Events

Use telemetry events for:

- Authorization allow/deny outcomes at the write authorization boundary.
- Override outcomes (for example platform `ADMIN` break-glass) when present.
- Internal authorization evaluation failures (dependency/read errors).

## Canonical Telemetry Event

- `event_name`: `telemetry.authz.decision`

## Package Ownership

- `telemetry.authz.decision` and related game-facing audit events are now owned by the game service in `internal/services/game/observability/audit`.
- Runtime emission points are the game gRPC interceptor (`GRPCRead`/`GRPCWrite`) and game authorization policy helpers (`AuthzDecision`).
- Stable event names remain `telemetry.*` for downstream dashboard compatibility while the owning package is now game-specific.

Required attributes:

- `decision`: `allow` | `deny` | `override`
- `reason_code`: stable machine-readable reason code
- `policy_action`: normalized policy action label
- `grpc_code`: gRPC status code (`OK`, `PermissionDenied`, etc.)

Conditional required attributes:

- `override_reason`: non-empty when `decision=override`

Recommended attributes:

- `campaign_access`: resolved actor campaign access (`owner|manager|member`)
- `actor_user_id`: resolved user identifier when available
- `character_id`, `participant_id`, `target_*`: target resource identifiers when
  applicable
- `participant_operation`: participant-governance operation label when supplied
- `target_owns_active_characters`: boolean invariant signal for participant
  removal checks

Envelope fields should always include:

- `campaign_id`
- `actor_type`
- `actor_id`
- `request_id`
- `invocation_id`
- `trace_id`
- `span_id`
- `timestamp`

## Reason Code Policy

Reason codes must be stable and backward-compatible for dashboards and alert
rules.

Current baseline set:

- `AUTHZ_ALLOW_ACCESS_LEVEL`
- `AUTHZ_ALLOW_ADMIN_OVERRIDE`
- `AUTHZ_ALLOW_RESOURCE_OWNER`
- `AUTHZ_DENY_ACCESS_LEVEL_REQUIRED`
- `AUTHZ_DENY_MISSING_IDENTITY`
- `AUTHZ_DENY_ACTOR_NOT_FOUND`
- `AUTHZ_DENY_NOT_RESOURCE_OWNER`
- `AUTHZ_DENY_TARGET_IS_OWNER`
- `AUTHZ_DENY_LAST_OWNER_GUARD`
- `AUTHZ_DENY_MANAGER_OWNER_MUTATION_FORBIDDEN`
- `AUTHZ_DENY_TARGET_OWNS_ACTIVE_CHARACTERS`
- `AUTHZ_ERROR_DEPENDENCY_UNAVAILABLE`
- `AUTHZ_ERROR_ACTOR_LOAD`
- `AUTHZ_ERROR_OWNER_RESOLUTION`

Runtime override signaling (game gRPC write auth boundary):

- `x-fracturing-space-platform-role: ADMIN`
- `x-fracturing-space-authz-override-reason: <non-empty reason>`

## Query and Operations Guidance

Primary operational queries:

- Deny rate by `policy_action` and `reason_code`.
- Internal-error rate (`reason_code` prefixed `AUTHZ_ERROR_`).
- Top actors and campaigns by deny count.
- Override usage counts and reasons (when override paths are enabled).

Operational recommendations:

- Alert on sustained spikes in `AUTHZ_ERROR_*`.
- Track rollout by monitoring deny distributions before/after policy changes.
- Keep dashboards keyed by `reason_code` and `policy_action`, not free-text
  messages.
