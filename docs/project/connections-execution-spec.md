---
title: "Connections Execution Spec"
parent: "Project"
nav_order: 20
---

# Connections Execution Spec

## Purpose

Define an execution-ready plan for the remaining Connections roadmap work and
resolve how existing `auth` account profile data fits with `connections`
username/public-profile data.

As of **2026-02-22**, this spec is the implementation guide for unfinished
Connections milestones.

This boundary is based on domain semantics (what a field is for), not where
legacy storage happened to exist first.

## Current State Snapshot (2026-02-22)

Implemented:

- Phase 1 contacts boundary cutover (`AddContact`, `RemoveContact`,
  `ListContacts`) in `connections`.
- Phase 2 username behavior in `connections` (`SetUsername`, `GetUsername`,
  `LookupUsername`) and web invite username resolution.
- Connections public profile APIs and storage (`SetPublicProfile`,
  `GetPublicProfile`, `LookupPublicProfile`).
- Web invite username verification context powered by
  `LookupPublicProfile`.
- Roadmap/phase status language updated to mark Phase 3 as partially
  implemented.

Not yet complete:

- Owner-managed web settings flow for editing and viewing the connections public
  profile (`name`, `avatar_set_id`, `avatar_asset_id`, `bio`).
- Connections API reference coverage in `docs/reference/`.
- Phase 4 contact-link permalink spec and implementation.

## Boundary Rubric (AuthN/AuthZ vs Social)

Use this rule for all future fields and APIs:

1. If data proves identity, authenticates a user, or grants/denies access, it
   belongs to `auth`.
2. If data helps users find, recognize, or relate to other users, it belongs to
   `connections`.
3. If data is a private account preference (for example locale), it is not
   social data and should not be owned by `connections`.
4. Field ownership is single-writer per field; web may compose read models but
   must not create hidden cross-service write replication.

## Boundary Model: Auth Domain vs Connections Domain

The system intentionally keeps two profile surfaces with different owners and
purposes.

### Auth domain (owner: `auth`)

Source of truth for authN/authZ primitives:

- user principal and recovery channels (user/email/passkey/magic link)
- auth session and OAuth token issuance/verification
- authorization artifacts such as join-grant issuance

Canonical APIs:

- `auth.v1.AuthService` operations
- auth user record read/write operations for private account settings

Primary usage:

- login/recovery/session/token workflows
- account `/profile` private settings UX

Clean-state private account settings in `auth`:

- `locale` (stored on the auth user record, not a separate profile table)

### Clean-state field ownership for ambiguous profile fields

- `name` -> `connections`
- `locale` -> `auth`
- `avatar_set_id` -> `connections`
- `avatar_asset_id` -> `connections`

### Connections domain (owner: `connections`)

Source of truth for social/discovery identity and relationship context:

- `username`
- `name`
- `avatar_set_id`
- `avatar_asset_id`
- `bio`

Canonical APIs:

- `connections.v1.ConnectionsService.SetUsername`
- `connections.v1.ConnectionsService.GetUsername`
- `connections.v1.ConnectionsService.LookupUsername`
- `connections.v1.ConnectionsService.SetPublicProfile`
- `connections.v1.ConnectionsService.GetPublicProfile`
- `connections.v1.ConnectionsService.LookupPublicProfile`

Primary usage:

- Invite recipient targeting (`@username` -> `recipient_user_id`).
- Invite verification context before submit.
- Future public discovery and relationship UX.

### Invariants

1. No shared write ownership for the same field across services.
2. `auth` does not write `connections` profile records.
3. `connections` does not write `auth` account profile records.
4. `locale` remains a private account setting owned by `auth`.
5. `name` and avatar identity fields are social/discovery metadata owned by
   `connections`.
6. Web composes both services at read time instead of write-through syncing.
7. `game` invite authority remains unchanged (`recipient_user_id` write target).

## Composition Rules in Web

1. `/profile` remains private account settings backed by auth-owned user data.
2. `/settings/username` remains backed by `connections` username APIs.
3. Add `/settings/public-profile` backed by `connections` public profile APIs.
4. Invite verification continues using `connections.LookupPublicProfile`.
5. No implicit cross-service replication on save actions.

## Execution Milestones

### Milestone A: Web owner-managed public profile settings

Deliverables:

- Add routes/handlers/templates for reading and writing connections public
  profile in web settings.
- Reuse existing validation and status mapping from `connections` service.
- Keep `/profile` (auth/private settings) and public-profile settings
  (connections/social settings) as separate UI surfaces.

Acceptance checks:

- Authenticated user can create/update public profile in settings.
- Missing profile returns a clean empty-state form and does not break settings.
- Invalid input surfaces friendly errors mapped from gRPC status.
- Existing username settings and invite verification flows do not regress.

### Milestone B: Connections API reference coverage

Deliverables:

- Add explicit reference docs for connections gRPC methods and error semantics.

Acceptance checks:

- Connections APIs are discoverable from `docs/reference/` without reading proto
  files directly.

### Milestone C: Phase 4 contact link permalinks

Deliverables:

- Create Phase 4 spec doc (domain model, APIs, lifecycle, security constraints).
- Implement create/revoke/consume permalink flow in `connections`.
- Add web flow for link generation and consumption.

Acceptance checks:

- Link lifecycle operations are auditable and idempotent.
- Consuming a valid link creates one directed contact edge.
- Replayed/expired/revoked links fail with deterministic status codes.

## Open Decisions

1. Should contact requests (`pending/accepted/declined`) be introduced before
Phase 4 permalinks?
Recommended: resolve before implementation start for Phase 4.

## Non-goals for This Spec

- Collapsing auth and connections profile stores into one service.
- Moving invite authority from `game` into `connections`.
- Introducing fuzzy search or recommendation ranking.
