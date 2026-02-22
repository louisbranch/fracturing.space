---
title: "Connections Phase 2 Username"
parent: "Project"
nav_order: 18
---

# Connections Phase 2: Username Claim and Lookup

## Purpose

Define an implementation-ready specification for the next `connections` phase:

- per-user username claim/update,
- deterministic normalization + uniqueness,
- lookup by username for discovery and invite targeting.

Phase 2 goal:

- users can discover and target another user by a stable username without
  requiring internal user IDs in UX flows.

## Scope and Non-goals

In scope:

- `connections.v1.ConnectionsService` username operations.
- Username normalization and uniqueness contract.
- Exact username lookup for discovery.
- Storage/index additions in `connections` SQLite database.
- Web invite targeting by username via `@username` resolution (resolve username
  -> `user_id`, then keep `game` invite APIs unchanged).

Out of scope:

- Public profile fields (display name/avatar/bio), deferred to Phase 3.
- Prefix/fuzzy search and social graph recommendations.
- Contact request workflow (`pending/accepted/declined`).
- Contact link permalinks (Phase 4).
- New anti-abuse/rate-limit policy changes beyond existing transport controls.

## Service Boundary

Boundary ownership remains:

- `connections`: username state, normalization policy, lookup index, directed
  contacts.
- `auth`: user identity authority, authentication, sessions, OAuth, join grant
  issuance.
- `game`: invite lifecycle and seat claim enforcement.
- `web`: orchestrates user-facing forms and API composition.

Phase 2 does not move invite authority into `connections`.

## Domain Model

### UsernameRecord

A username record is identity metadata owned by `connections`:

- `user_id` (owner user)
- `username` (canonical normalized username)
- `created_at`
- `updated_at`

Invariants:

1. One user has at most one active username.
2. One username maps to exactly one user.
3. Username uniqueness is global and case-insensitive (enforced via canonical
   form).
4. Username updates are atomic: claim new value and release old value in one
   operation.
5. Re-setting the same canonical username for the same user is idempotent.
6. Username lookup returns minimal identity reference (`user_id` + `username`)
   only in Phase 2.
7. Contacts remain directed edges by `user_id`; username is a lookup aid, not a
   replacement key for contact storage.

## Normalization and Validation Contract

Canonicalization pipeline:

1. Trim surrounding whitespace.
2. Reject if empty.
3. Convert to lowercase ASCII.
4. Validate against `^[a-z][a-z0-9._-]{2,31}$` (3-32 chars, starts with a
   letter).

Phase 2 intentionally supports ASCII-only usernames to keep matching rules and
storage indexes deterministic. Broader Unicode handling can be considered in a
later phase.

Validation failures return `invalid_argument`.

## API Surface (Planning)

Additions to `connections.v1.ConnectionsService`:

- `SetUsername`
  - request: `user_id`, `username`
  - behavior: create or update the caller's username record
  - response: `UsernameRecord`
- `GetUsername`
  - request: `user_id`
  - behavior: fetch username for one user
  - response: `UsernameRecord`
- `LookupUsername`
  - request: `username`
  - behavior: resolve username to owning user
  - response: `UsernameRecord`

Recommended shared message:

- `UsernameRecord`
  - `user_id`, `username`, `created_at`, `updated_at`

Compatibility notes:

- Existing `AddContact`/`RemoveContact`/`ListContacts` APIs remain unchanged.
- Invite writes continue to pass `recipient_user_id` to `game`; web resolves
  that ID from username via `connections.LookupUsername` when input is prefixed
  as `@username`. Direct `recipient_user_id` input remains supported.

## Error Taxonomy

Canonical status mapping:

- `invalid_argument`: malformed `user_id`/`username` or failed normalization.
- `already_exists`: requested username is claimed by a different user.
- `not_found`: username lookup miss or user has no claimed username.
- `internal`: storage or invariant enforcement failure.
- `unavailable`: transient dependency/storage availability issue.

Idempotency behavior:

- `SetUsername` with the same canonical username for the same user returns
  success and current record.

## Storage and Migration

Phase 2 migration adds a username table in `connections` storage (example name:
`usernames`):

- `user_id TEXT PRIMARY KEY`
- `username TEXT NOT NULL UNIQUE`
- `created_at INTEGER NOT NULL`
- `updated_at INTEGER NOT NULL`

Indexing:

- Unique index on canonical `username` (or inline `UNIQUE`) to enforce global
  uniqueness.

No backfill is required from `auth`: there is no existing canonical username
field to migrate.

## Rollout Sequence

1. Add `connections` migration + storage methods + gRPC endpoints.
2. Add/adjust web settings flow to claim/update username via `SetUsername`.
3. Add invite form path for `username -> user_id` resolution before calling
   invite create.
4. Keep existing user-id invite path as fallback during rollout.
5. Add integration tests for claim/update/lookup and invite-by-username flow.

## Acceptance Checks

- A user can claim an available valid username.
- A user can update their username; old username becomes available.
- Duplicate claim by another user returns `already_exists`.
- Lookups are case-insensitive through canonicalization.
- Invalid usernames fail with `invalid_argument`.
- `GetUsername` and `LookupUsername` return `not_found` when missing.
- Existing contact list/add/remove behavior does not regress.
- Web invite flow can resolve and submit `recipient_user_id` from entered
  `@username`.

## Implementation Notes

- Keep username validation centralized in one helper to avoid drift between
  storage and service layers.
- Favor simple exact-match lookup in Phase 2; avoid introducing partial-search
  ranking semantics until profile/discovery expands.
- This phase resolves the roadmap question by placing invite-by-username in
  Phase 2 (resolution only), while richer identity verification remains a Phase
  3 profile concern.
