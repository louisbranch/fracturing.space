---
title: "Connections Phase 3 Public Profile"
parent: "Project"
nav_order: 19
---

# Connections Phase 3: Public Profile Surface and Username Verification

## Purpose

Define an implementation-ready specification for the next `connections` phase:

- owner-managed public profile fields (`display_name`, `avatar_url`, `bio`),
- lookup of public profile context by username,
- web invite verification context for `@username` targeting.

Phase 3 goal:

- users can verify identity context for a username before inviting or
  connecting.

## Scope and Non-goals

In scope:

- `connections.v1.ConnectionsService` public profile operations.
- Storage additions in `connections` SQLite database for public profile fields.
- Username-based profile lookup for verification context.
- Web invite verification UI for `@username` targets.

Out of scope:

- Fuzzy/prefix discovery and recommendation ranking.
- Privacy settings beyond fixed public profile visibility.
- Contact request workflow (`pending/accepted/declined`).
- Contact permalink lifecycle (Phase 4).
- Anti-abuse policy redesign beyond existing transport/rate controls.

## Service Boundary

Boundary ownership remains:

- `connections`: usernames, public profile metadata, directed contacts.
- `auth`: identity authority, authn/authz, sessions, OAuth.
- `game`: invite lifecycle and seat claim enforcement.
- `web`: user-facing composition of `connections` and `game` APIs.

Phase 3 does not move invite authority into `connections`.

## Domain Model

### PublicProfileRecord

A public profile record is metadata owned by `connections`:

- `user_id` (profile owner),
- `display_name`,
- `avatar_url`,
- `bio`,
- `created_at`,
- `updated_at`.

Invariants:

1. One user has at most one public profile record.
2. Public profile fields are mutable by the owning user only.
3. Username lookup remains authoritative for identity targeting; profile is
   verification context, not a replacement identifier.
4. Public profile lookup by username can return username identity even when
   profile fields are absent.

## Validation Contract

`SetPublicProfile` field rules:

- `display_name`: required after trimming, max 64 chars.
- `avatar_url`: optional; max 512 chars; when set must be absolute `http://` or
  `https://` URL.
- `bio`: optional; max 280 chars.

Validation failures return `invalid_argument`.

## API Surface (Planning)

Additions to `connections.v1.ConnectionsService`:

- `SetPublicProfile`
  - request: `user_id`, `display_name`, `avatar_url`, `bio`
  - behavior: create or update one owner's public profile record
  - response: `PublicProfileRecord`
- `GetPublicProfile`
  - request: `user_id`
  - behavior: fetch profile by owner user ID
  - response: `PublicProfileRecord`
- `LookupPublicProfile`
  - request: `username`
  - behavior: resolve username to owner and return verification context
  - response: `UsernameRecord` plus optional `PublicProfileRecord`

Recommended shared message:

- `PublicProfileRecord`
  - `user_id`, `display_name`, `avatar_url`, `bio`, `created_at`, `updated_at`

Compatibility notes:

- Existing contact and username operations remain unchanged.
- Invite create continues sending `recipient_user_id` to `game`.
- Web may call `LookupPublicProfile` for verification and still use
  `recipient_user_id` for invite writes.

## Error Taxonomy

Canonical status mapping:

- `invalid_argument`: malformed `user_id`, invalid username, or profile field
  validation failure.
- `already_exists`: reserved for future uniqueness constraints beyond username
  ownership (not expected in baseline profile flow).
- `not_found`: username miss or profile missing for direct `GetPublicProfile`.
- `internal`: storage or invariant enforcement failure.
- `unavailable`: transient dependency/storage availability issue.

Lookup semantics:

- `LookupPublicProfile` returns `not_found` when username is missing.
- `LookupPublicProfile` returns success with `username_record` and optional
  `public_profile` when username exists.

## Storage and Migration

Phase 3 migration adds a `public_profiles` table in `connections` storage:

- `user_id TEXT PRIMARY KEY`
- `display_name TEXT NOT NULL`
- `avatar_url TEXT NOT NULL DEFAULT ''`
- `bio TEXT NOT NULL DEFAULT ''`
- `created_at INTEGER NOT NULL`
- `updated_at INTEGER NOT NULL`

No backfill is required from other services.

## Rollout Sequence

1. Add migration + storage methods + gRPC endpoints for public profiles.
2. Add/adjust web settings flow for owner profile write/read.
3. Add web invite verification for `@username` profile context.
4. Keep invite writes on `game` with `recipient_user_id`.
5. Add tests for profile create/update/get/lookup and invite verification flow.

## Acceptance Checks

- Owner can set and update a valid public profile.
- `GetPublicProfile` returns `not_found` when profile is absent.
- `LookupPublicProfile` resolves username and returns verification context.
- Invalid profile input returns `invalid_argument`.
- Existing contact and username behavior does not regress.
- Web invite flow can show verification context for entered `@username`.

## Implementation Notes

- Keep profile validation in one helper to avoid drift between storage and
  service layers.
- Keep lookup deterministic and exact in this phase; no ranking or fuzzy search.
- Do not couple profile schema to auth internals; `connections` owns the public
  representation.
