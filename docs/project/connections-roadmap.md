---
title: "User Connections Roadmap"
parent: "Project"
nav_order: 17
---

# User Connections Roadmap

As of 2026-02-22, this page is an active roadmap artifact (not a canonical
onboarding entrypoint).

Canonical runtime context for this roadmap:

- [Architecture](architecture.md)
- [Domain language](domain-language.md)
- [Identity and Recovery](identity.md)

## Purpose

Define a phased path for user discovery and contact UX while keeping `auth` focused on authentication and authorization.

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

## Ownership Split

- `auth`: user identity, passkeys, sessions, OAuth, join grant issuance, and private account settings on the user record (for example locale).
- `connections`: contact graph and social identity fields in one `UserProfile` aggregate (`username`, `name`, `avatar_set_id`, `avatar_asset_id`, `bio`), plus future connection request lifecycle and connection links.
- `game`: invite lifecycle and campaign seat claim enforcement.
- `notifications`: inbox intents and delivery status.
- `web`: UX orchestration across service APIs.

## Phase 1 (Completed): Stand Up `connections` + Migrate Contacts

### Goals

- Create a new `connections` service boundary.
- Move contact ownership from `auth` to `connections`.
- Preserve existing user-visible behavior.

### Status

Completed.

## Phase 2 (Completed): Unified `UserProfile`

### Scope

- One user-owned social/discovery profile aggregate in `connections`.
- One API family for write/read/lookup:
  - `SetUserProfile`
  - `GetUserProfile`
  - `LookupUserProfile`
- Canonical username normalization + uniqueness as part of the profile aggregate.
- Invite recipient resolution and verification via `LookupUserProfile`.

### Primary Outcome

A user can claim a stable username and publish discoverable identity context from one profile record without split username/public-profile entities.

### Status (as of 2026-02-22)

Completed.

## Phase 3 (Planned): Contact Link Permalink

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
