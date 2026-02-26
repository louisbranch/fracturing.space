---
title: "Social Execution Spec"
parent: "Project"
nav_order: 20
---

# Social Execution Spec

## Purpose

Define the execution model for `social` social/discovery behavior with a
single `UserProfile` aggregate.

As of **2026-02-22**, this spec reflects a breaking refactor that removed the
split `username` and `public profile` entities.

## Current State Snapshot (2026-02-22)

Implemented:

- Contacts boundary cutover (`AddContact`, `RemoveContact`, `ListContacts`) in `social`.
- Unified profile APIs and storage in `social`:
  - `SetUserProfile`
  - `GetUserProfile`
  - `LookupUserProfile`
- Web invite recipient resolution and verification context both use
  `LookupUserProfile`.
- Web settings profile route is backed by unified profile APIs.

## Boundary Rubric (AuthN/AuthZ vs Social)

Use this rule for all future fields and APIs:

1. If data proves identity, authenticates a user, or grants/denies access, it
   belongs to `auth`.
2. If data helps users find, recognize, or relate to other users, it belongs to
   `social`.
3. If data is a private account preference (for example locale), it is not
   social data and should not be owned by `social`.
4. Field ownership is single-writer per field; web may compose read models but
   must not create hidden cross-service write replication.

## Boundary Model: Auth Domain vs Social Domain

### Auth domain (owner: `auth`)

Source of truth for authN/authZ primitives:

- user principal and recovery channels (user/email/passkey/magic link)
- auth session and OAuth token issuance/verification
- authorization artifacts such as join-grant issuance

Clean-state private account settings in `auth`:

- `locale`

### Social domain (owner: `social`)

Source of truth for social/discovery identity and relationship context via one
aggregate:

- `user_id`
- `username` (canonical unique)
- `name`
- `avatar_set_id`
- `avatar_asset_id`
- `bio`

Canonical APIs:

- `social.v1.SocialService.SetUserProfile`
- `social.v1.SocialService.GetUserProfile`
- `social.v1.SocialService.LookupUserProfile`

Primary usage:

- Invite recipient targeting (`@username` -> `recipient_user_id`).
- Invite verification context before submit.
- Discovery and relationship UX.

## Invariants

1. No shared write ownership for the same field across services.
2. `auth` does not write `social` profile records.
3. `social` does not write `auth` private preference fields (for example `locale`).
4. `locale` remains a private account setting owned by `auth`.
5. Social/discovery profile identity fields are owned by `social`.
6. Web composes services at read time instead of write-through syncing.
7. `game` invite authority remains unchanged (`recipient_user_id` write target).

## Composition Rules in Web

1. `/app/profile` remains private account settings backed by auth-owned locale data only.
2. `/settings/user-profile` is backed by `social` unified profile APIs.
3. Invite verification uses `social.LookupUserProfile`.
4. No implicit cross-service replication on save actions.

## Next Milestones

### Milestone A: Social API reference coverage

Deliverables:

- Add explicit reference docs for social gRPC methods and error semantics.

### Milestone B: Contact link permalinks

Deliverables:

- Define and implement create/revoke/consume permalink flow in `social`.
- Add web flow for link generation and consumption.

## Non-goals for This Spec

- Collapsing auth and social into a single service.
- Moving invite authority from `game` into `social`.
- Introducing fuzzy search or recommendation ranking.
