# Session 15: Storage Layer — SQLite Implementations

## Status: `complete`

## Package Summaries

### `storage/sqlite/coreprojection/` (~41 files)
Largest storage package. Implements storage contracts for all core entity projections (campaign, participant, character, invite, session, scene, interaction, etc.)

### `storage/sqlite/eventjournal/` (8 files, store_events.go at 908 lines)
Event journal implementation: append, read, replay. The core persistence mechanism.

### `storage/sqlite/projectionapplyoutbox/` (store.go at 631 lines)
Projection apply outbox: tracks which events have been projected.

### `storage/sqlite/integrationoutbox/` (store.go at 682 lines)
Integration outbox: tracks which events need external notification delivery.

## Findings

### Finding 1: coreprojection/ at 41 Files — Split by Entity
- **Severity**: high
- **Location**: `storage/sqlite/coreprojection/`
- **Issue**: 41 files in one package. This is a god package that implements storage for all core entities: campaigns, participants, characters, invites, sessions, scenes, interactions, events, spotlights, gates, etc. A contributor working on session storage must navigate past campaign and character files.
- **Recommendation**: Split by entity or concern:
  - `storage/sqlite/campaignstore/`
  - `storage/sqlite/participantstore/`
  - `storage/sqlite/sessionstore/`
  - etc.
  Each sub-package implements the corresponding `storage.XxxStore` interface. This matches the contract structure.

### Finding 2: store_events.go at 908 Lines — Decomposition Candidate
- **Severity**: high
- **Location**: `storage/sqlite/eventjournal/store_events.go`
- **Issue**: 908 lines in one file handling event persistence, querying, replay, and integrity operations. This is the most critical code path in the system (event sourcing journal). Multiple concerns in one file increases the risk of accidental regressions.
- **Recommendation**: Split into: `store_append.go` (write path), `store_query.go` (read path), `store_replay.go` (replay iteration), `store_integrity.go` (hash/chain verification).

### Finding 3: Outbox Implementations at 631/682 Lines — Large but Focused
- **Severity**: medium
- **Location**: `storage/sqlite/projectionapplyoutbox/store.go` (631 lines), `storage/sqlite/integrationoutbox/store.go` (682 lines)
- **Issue**: Both outbox implementations handle: outbox entry tracking, batch delivery, retry logic, and cleanup. The size is proportional to the responsibility, but 600+ lines in one file makes navigation harder.
- **Recommendation**: Split each into `store_write.go` (append entries) and `store_read.go` (delivery queries, batch management).

### Finding 4: N+1 Query Risk in List Operations
- **Severity**: medium
- **Location**: `storage/sqlite/coreprojection/`
- **Issue**: List operations (e.g., ListParticipantsByCampaign, ListCharacters) should use batch SQL queries rather than per-record lookups. The sqlc-generated queries likely handle this, but verify that JOIN operations are used rather than per-row subqueries.
- **Recommendation**: Audit the generated SQL queries for N+1 patterns. Ensure list operations use single queries with appropriate JOINs or IN clauses.

### Finding 5: Transaction Boundary Correctness
- **Severity**: info
- **Location**: `storage/sqlite/eventjournal/store_events.go`
- **Issue**: `BatchAppend` uses a transaction to persist all events from a single command atomically. This is critical for event sourcing correctness — partial appends would corrupt the event stream.
- **Recommendation**: Verify that the transaction wraps the entire batch, not individual events.

## Summary Statistics
- Files reviewed: ~72 (41 coreprojection + 8 eventjournal + projectionapplyoutbox + integrationoutbox)
- Findings: 5 (0 critical, 2 high, 2 medium, 0 low, 1 info)
