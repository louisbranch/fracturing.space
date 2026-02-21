# Persist logged-in web sessions across web server restarts

This plan documents how to use the web SQLite persistence path (`CacheDBPath`) to keep users logged in after a process restart without changing the token model or admin/auth protocols.

## Purpose / Big Picture

Web sessions are currently process-local (`sessionStore` only keeps an in-memory `map[sessionID]*session`). On restart, every in-flight `fs_session` cookie is invalid, so users are pushed to login even if their `fs_token` and token TTL are still valid.

This plan makes restart behavior resilient by persisting the web session mapping in the existing web DB and restoring it at request time. The goal is to keep the user experience stable while preserving current authorization boundaries (`sessionFromRequest` → campaign checks → token introspection / metadata propagation).

## Progress

- [x] (2026-02-20) Plan drafted and reviewed with scope; no production files changed yet.
- [x] (2026-02-20) Added implementation plan with persistence + fallback behavior.
- [x] (2026-02-20) Add DB migration + storage methods for durable web sessions.
- [x] (2026-02-20) Refactor `sessionStore` to hydrate from SQLite when session is absent from memory and persist on create/delete.
- [x] (2026-02-20) Add restart/invalidation tests proving valid `fs_session` survives process reuse and expired/missing records do not.
- [x] Validate behavior with focused unit/integration tests and document any residual risks (tests added, validated on 2026-02-20 via `make integration`).

## Surprises & Discoveries

- There is already a web SQLite cache DB at `data/web-cache.db` with migrations and lifecycle handling.
- `fs_token` (domain-scoped access token) is already a second cookie and remains valid independently of restart if the token TTL is still active.
- The current `session` struct includes cached derived values (`cachedUserID`, `cachedUserAvatar`) that are only process-local and can be recomputed after restore.
- Existing tests already assert session cookie creation and deletion and cache-store initialization, so restart coverage can be added near `session_test.go` and `server_test.go`.

## Decision Log

- Decision: Use the existing web cache DB (`internal/services/web/storage/sqlite`) for session persistence instead of introducing a new DSN.
  - Rationale: reuse existing migration/bootstrapping path and reduce operational complexity.
  - Date/Author: 2026-02-20 / session
- Decision: Implement a two-layer session storage strategy: in-memory hot cache plus SQLite durable store.
  - Rationale: keep current low-latency behavior and provide restart recovery by loading missing sessions from disk.
  - Date/Author: 2026-02-20 / session
- Decision: Keep auth logic unchanged (`sessionFromRequest` remains the gate) and only add robust fallback from disk.
  - Rationale: avoids altering campaign/user authorization semantics while improving continuity.
  - Date/Author: 2026-02-20 / session
- Decision: Add compatibility fallback for missing disk session rows by optionally rebuilding a short-lived session from `fs_token` and immediate introspection when needed.
  - Rationale: avoids accidental lockout during migration and handles partial writes/corruption gracefully.
  - Date/Author: 2026-02-02 / session

## Outcomes & Retrospective

- Deferred until implementation and validation are complete.

## Context and Orientation

- Session lifecycle entry points:
  - `internal/services/web/session.go`
  - `internal/services/web/server.go` (`handleAuthCallback`, `handleAuthLogout`, `sessionFromRequest` call sites)
  - `internal/services/web/game_layout.go` and `campaigns.go` for cached user context usage
- Auth + identity resolution dependencies:
  - `internal/services/web/campaign_access.go` (`introspectUserID`)
  - `internal/services/web/storage` (cache store bootstrap + SQLite migrations)

## Plan of Work

1. Establish durable session schema and storage API in the web storage layer.
2. Refactor web session retrieval/creation/deletion to consult persistence as a fallback.
3. Wire callback/logout paths to keep DB and in-memory state consistent.
4. Add tests that validate restart behavior and cleanup.
5. Confirm migration safety and backward compatibility on service restart.

## Concrete Steps

1. Add a `web_sessions` table migration under `internal/services/web/storage/sqlite/migrations` with fields:
   - `session_id TEXT PRIMARY KEY`
   - `access_token TEXT NOT NULL`
   - `display_name TEXT NOT NULL DEFAULT ''`
   - `expires_at INTEGER NOT NULL`
   - optional `created_at INTEGER NOT NULL` for housekeeping
2. Add SQLite-backed session methods in `internal/services/web/storage/sqlite`:
   - write session row on create
   - read session row by `session_id` for restore
   - delete session row on logout/manual deletion
   - prune expired rows opportunistically (on read) and via optional startup cleanup
3. Extend web persistence contract or add local session persistence interface in web package:
   - Keep `sessionStore` API shape (`create`, `get`, `delete`) unchanged for callers.
   - Back sessionStore with memory-first + SQLite fallback.
   - On `create`, write both memory + DB.
   - On `get`, check memory then DB; restore into memory and return.
   - On `delete`, remove from both layers.
4. Update auth callbacks and logout flow:
   - `handleAuthCallback` should persist session metadata and retain existing cookie writes.
   - `handleAuthLogout` should clear token/session in both in-memory and DB, then clear cookies.
5. Add migration + unit coverage:
   - `internal/services/web/storage/sqlite/store_test.go` assert `web_sessions` table exists and session CRUD behavior is durable across reopened store.
   - `internal/services/web/session_test.go` add restart-style tests:
     - create in one store and read from another instance sharing DB path
     - expired session rows are rejected and removed
     - logout delete clears durable row
   - `internal/services/web/server_test.go` add request-level behavior test:
     - login callback persists session row
     - recreated handler with same `CacheDBPath` + same `fs_session` cookie can read user session
     - logout after restart simulation removes DB row and rerendered request falls back to unauthenticated path
6. Run behavior change validation in TDD sequence:
   - Red: first add at least one failing test per behavior.
   - Green: implement minimum changes until tests pass.
   - Refactor: only if behavior unchanged and tests remain green.
   - Log exact command and file for each Red/Green step in PR notes.

## Validation and Acceptance

- Required command sequence for behavior confirmation:
  - `go test ./internal/services/web/storage/sqlite -run TestOpenRunsMigrations`
  - `go test ./internal/services/web -run SessionStore|AuthCallback|AuthLogout`
  - Restart-simulation test can be covered by handler-level test cases that instantiate two `handler`/`sessionStore` objects against the same DB path and `fs_session` cookie.
- Acceptance checks:
  - A user with a fresh login can restart web service without being redirected to `/auth/login` while their access token is still valid.
  - Expired sessions are rejected even if row is present on disk.
  - Logout always invalidates both cookie and persistent state.

## Idempotence and Recovery

- Database migration must be idempotent and safe to re-run.
- Session row upserts should avoid duplicate insertion failures.
- Expired rows must be safe to ignore/clear on read.
- If DB unavailable or schema missing at runtime, web should degrade gracefully to existing in-memory behavior and proceed if `fs_session` can still be rebuilt through fallback logic.

## Artifacts and Notes

- This plan file is the source of implementation truth.
- After implementation, consider adding a short architecture note under `docs/project/web-capability-parity.md` if operational behavior (restart resilience) changes.

## Interfaces and Dependencies

- `handler` currently uses `*sessionStore`; session persistence should remain encapsulated there so route handlers can stay unchanged.
- Web persistence for auth continuity will depend on:
  - `internal/services/web/storage/sqlite`
  - `internal/services/web/storage/sqlite/migrations`
  - existing `CacheDBPath` startup flow in `internal/services/web/server.go`
