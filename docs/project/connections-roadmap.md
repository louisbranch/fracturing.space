---
title: "User Connections Roadmap"
parent: "Project"
nav_order: 17
---

# User Connections Roadmap

## Purpose

Define a phased path for user discovery and contact UX while keeping `auth` focused on authentication and authorization.

## Problem Statement

Current behavior supports:

- Directed contacts in `auth` (`AddContact`, `RemoveContact`, `ListContacts`).
- Campaign invite claim flows in `game`, with optional `recipient_user_id`.

Missing behavior is a clear, bounded domain for user-to-user discovery and connection workflows. Keeping this in `auth` risks service sprawl.

## Domain Boundary Decision

Use a dedicated `connections` service boundary (not `people`).

- Canonical entity term remains `User`.
- `auth` remains authN/authZ focused (proof of identity and access artifacts).
- `connections` owns social/discovery and relationship metadata.

Boundary rubric:

1. AuthN/authZ artifacts belong to `auth`.
2. User discovery/recognition/relationship artifacts belong to `connections`.
3. Private account preferences are not social artifacts and do not belong to
   `connections` by default.

### Ownership Split

- `auth`: user identity, passkeys, sessions, OAuth, join grant issuance, and private account settings on the user record (for example locale).
- `connections`: contact graph and social identity fields (`username`, `name`, `avatar_set_id`, `avatar_asset_id`, `bio`), plus future connection request lifecycle and connection links.
- `game`: invite lifecycle and campaign seat claim enforcement.
- `notifications`: inbox intents and delivery status.
- `web`: UX orchestration across service APIs.

## Phase 1 (Completed): Stand Up `connections` + Migrate Contacts

### Goals

- Create a new `connections` service boundary.
- Move contact ownership from `auth` to `connections`.
- Preserve existing user-visible behavior.

### In Scope

- New `connections.v1.ConnectionsService` with contact endpoints:
  - `AddContact`
  - `RemoveContact`
  - `ListContacts`
- New `connections` storage for directed contacts.
- Migration of existing contact rows from `auth` storage to `connections` storage.
- Clean-slate cutover to `connections` contacts APIs.

### Out of Scope

- Username claim/search.
- Public profile discovery.
- Contact request state machine.
- Contact permalinks.
- New privacy or anti-abuse controls.

### Cutover Strategy

Clean-slate move with no compatibility bridge:

1. Switch contact writes/reads to `connections` endpoints.
2. Remove contact handling from `auth` in the same delivery window.
3. Keep behavior and pagination semantics, but not old endpoint compatibility.

### Data Migration Strategy

1. Introduce `connections` contact table and API.
2. Run one-time migration from `auth.user_contacts` into `connections` storage.
3. Verify row counts and spot-check owner/contact pairs.
4. Cut clients to `connections`.
5. Remove old `auth` contact storage path as part of the same cutover.

### Phase 1 Acceptance Criteria

- Contacts are served from `connections`, not `auth`.
- Existing add/list/remove contact behavior matches current semantics.
- No regression in campaign invite flow.
- `auth` no longer exposes or owns contact behavior.

## Phase 2 (Implemented): Username

### Scope

- Per-user unique username claim/update.
- Username normalization and uniqueness rules.
- Username lookup for discovery.

### Primary Outcome

A user can find another user by username without needing internal user IDs.

Detailed phase spec:
[Connections Phase 2: Username Claim and Lookup](connections-phase2-username.md)

Status:

- Username claim/update and lookup APIs are implemented in `connections`.
- Invite `@username` resolution is implemented in `web` before submit to `game`.
- Settings username claim/update flow is implemented in `web`.

## Phase 3 (In Progress): Public Profile

### Scope

- Public profile surface (display name/avatar/bio subset).
- Profile lookup by username.

### Primary Outcome

Users can verify identity context before connecting/inviting.

Detailed phase spec:
[Connections Phase 3: Public Profile Surface and Username Verification](connections-phase3-public-profile.md)

Status:

- Public profile APIs and storage are implemented in `connections`.
- Web invite verification context via `LookupPublicProfile` is implemented.
- Remaining: owner-managed web settings flow for public profile write/read.
- Execution details and remaining milestones live in:
  [Connections Execution Spec](connections-execution-spec.md)

## Phase 4 (Planned): Contact Link Permalink

### Scope

- Create/revoke/consume durable contact link.
- Link consumption creates directed contact edge.

### Primary Outcome

A user can share one link that lets another user add them as a contact with minimal friction.

## Cross-Phase Invariants

- `User` is the domain identity term.
- Contacts are directed relationships, not mutual friendship by default.
- `game` invite authority and seat claim enforcement stay in `game`.
- Notifications are delivery mechanics, not source-of-truth state.

## Open Questions

- Should contact requests (pending/accepted/declined) be introduced before permalinks?
