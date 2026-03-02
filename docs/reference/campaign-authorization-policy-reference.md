---
title: "Campaign authorization policy reference"
parent: "Reference"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Campaign Authorization Policy Reference

Detailed policy semantics, target-context behavior, and rollout guidance for
campaign authorization.

For onboarding policy model, start with
[Campaign Authorization Model](../architecture/platform/campaign-authorization-model.md).

## Canonical implementation surface

Runtime policy source of truth lives in:

- `internal/services/game/domain/authz/policy.go`

Transport boundaries should call canonical evaluators rather than duplicate role
matrices in handlers.

## Detailed matrix notes

`Limited` manager behavior means:

- managers cannot assign or remove owner access
- managers cannot mutate owner targets
- managers can manage member records
- manager-to-manager operations are only allowed when policy explicitly permits

`Owned only` member behavior means:

- mutation allowed only when actor is current `RESOURCE_OWNER`

## Ownership projection contract

Ownership must be available in projection-backed character state for request-path
authorization checks.

Reference behaviors:

- owner defaults to creator on `character.created`
- ownership transfer uses explicit owner field mutation
- participant removal guard checks active owned resources

## `AuthorizationService.Can` target semantics

`AuthorizationService.Can` accepts optional `AuthorizationTarget` context.

Participant governance behavior:

1. `target_participant_id` (or `resource_id`) identifies target.
2. `target_campaign_access` may be supplied directly or resolved server-side.
3. `requested_campaign_access` activates invariant checks for access changes.
4. manager-owner mutation attempts are denied with stable reason codes.
5. final-owner demotion is denied when owner guard is triggered.
6. `participant_operation` labels disambiguate mutate/access-change/remove paths.

Character mutation behavior:

- ownership checks evaluate target owner directly or by resolving owner from
  projection state using `resource_id`

## Fork policy detail

- `ForkCampaign` is governance-scoped (`OWNER`/`MANAGER`/`ADMIN`).
- `GetLineage` remains read-scoped for campaign members.
- future open-fork starter-campaign policy is an explicit future decision, not
  implicit behavior.

## Batch authorization guidance

Use batched checks for UI visibility surfaces and per-row action states.

Contract:

- `BatchCan` accepts repeated checks with `check_id` correlation IDs.
- Responses preserve request order; clients should correlate by `check_id`.
- Batch and unary checks must share evaluator behavior and reason codes.
- Invalid batch payloads fail request (fail-fast).

## Web mutation gate guidance

- web mutation routes must require evaluated authorization decisions before
  gateway mutation calls
- fail closed on authz unavailability or unevaluated responses
- avoid participant-list or UI fallback approximations for mutation authority

## Override model

Override signaling is explicit and auditable.

Runtime headers:

- `x-fracturing-space-platform-role: ADMIN`
- `x-fracturing-space-authz-override-reason: <non-empty reason>`

Override usage must emit telemetry with clear reason attributes.

## Reason code surface

Stable machine-readable reason code families include:

- allow: `AUTHZ_ALLOW_*`
- deny: `AUTHZ_DENY_*`
- internal errors: `AUTHZ_ERROR_*`

Representative baseline codes:

- `AUTHZ_ALLOW_ACCESS_LEVEL`
- `AUTHZ_ALLOW_ADMIN_OVERRIDE`
- `AUTHZ_DENY_ACCESS_LEVEL_REQUIRED`
- `AUTHZ_DENY_TARGET_IS_OWNER`
- `AUTHZ_DENY_LAST_OWNER_GUARD`
- `AUTHZ_DENY_TARGET_OWNS_ACTIVE_CHARACTERS`
- `AUTHZ_ERROR_DEPENDENCY_UNAVAILABLE`

Telemetry contract details live in
[Campaign Authorization Audit and Telemetry](campaign-authorization-audit.md).

## Adoption and rollout sequencing

When changing authz policy behavior:

1. land policy change behind canonical evaluator updates
2. add or update reason-code tests
3. ensure web/mcp/grpc boundaries call same evaluator path
4. monitor deny/error distributions after rollout
5. tighten fallback/degraded behavior once telemetry confirms stability

## Related docs

- [Campaign Authorization Model](../architecture/platform/campaign-authorization-model.md)
- [Campaign Authorization Audit and Telemetry](campaign-authorization-audit.md)
- [Web architecture](../architecture/platform/web-architecture.md)
