---
title: "Campaign Authorization Model"
parent: "Platform surfaces"
nav_order: 1
status: canonical
owner: engineering
last_reviewed: "2026-03-03"
---

# Campaign Authorization Model

Concise architecture contract for campaign governance and gameplay mutation
authorization.

## Purpose

Define durable authorization boundaries for campaign-scoped operations. This
model is clean-slate; compatibility with legacy policy behavior is not required.

## Policy axes

- **Platform role**: `ADMIN`
- **Campaign access**: `OWNER`, `MANAGER`, `MEMBER`
- **Gameplay role**: `GM`, `PLAYER` (orthogonal to campaign governance)
- **Resource relationship**: `RESOURCE_OWNER`, `RESOURCE_CONTROLLER`, `SELF`

## Core rules

1. Server-side authorization is authoritative for all write actions.
2. Deny by default when no explicit allow rule exists.
3. Campaign governance decisions are based on access (`OWNER/MANAGER/MEMBER`).
4. Gameplay role labels do not implicitly grant governance rights.
5. `ADMIN` override requires explicit reason and audit telemetry.

## Permission summary

| Capability | ADMIN | OWNER | MANAGER | MEMBER |
| --- | --- | --- | --- | --- |
| Campaign reads | Allow | Allow | Allow | Allow |
| Campaign governance writes (metadata/settings/archive) | Allow | Allow | Allow | Deny |
| Participant governance (promote/demote/remove) | Allow | Allow | Limited | Deny |
| Invite create/revoke | Allow | Allow | Allow | Deny |
| Character create/update/delete | Allow | Allow | Allow | Owned only |
| Character ownership transfer | Allow | Allow | Deny | Deny |
| Session start/end and gate management | Allow | Allow | Allow | Deny |
| GM-only gameplay actions | Allow | Allow if GM | Allow if GM | Allow if GM |

`Limited` means managers cannot mutate owner access or violate final-owner
invariants.

## Invariants

1. A campaign always retains at least one `OWNER`.
2. Managers cannot assign or remove owner access.
3. Members cannot self-escalate campaign access.
4. Ownership transfer is explicit and audited.
5. Participant removal is blocked when active owned resources exist.
6. AI-controlled participants are restricted to `GM` + `MEMBER`, must not have a
   bound user identity, and cannot be rebound or seat-reassigned.

## Character ownership contract

- `owner_participant_id` controls governance authority.
- `controller_participant_id` controls operational gameplay use.
- Controller assignment does not transfer ownership.
- Member mutation rights are ownership-scoped unless elevated.

## Service boundary contract

- Runtime policy source of truth: `internal/services/game/domain/authz/policy.go`.
- Transport layers call canonical evaluators; they must not re-implement policy
  matrices ad hoc.
- Batch authorization checks must use the same evaluator and reason codes as
  unary checks.

## Deep references

- [Campaign authorization policy reference](../../reference/campaign-authorization-policy-reference.md)
- [Campaign authorization audit and telemetry](../../reference/campaign-authorization-audit.md)
