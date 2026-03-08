# GSR Phase 7: Storage Contracts & Implementations

## Summary

Sound foundational design with clear contract/implementation separation. Key issues: system-specific `storage_daggerheart.go` in core package, hidden type assertion coupling in `ProjectionApplyTxStore`, and `store_conversions.go` (933 lines) as a code generation opportunity. SQLite configuration is correct; transaction semantics are sound but underdocumented.

## Findings

### F7.1: `storage_daggerheart.go` in Core Package — Important

**Severity:** important

**Location:** `storage/storage_daggerheart.go` (553 lines)

14 interfaces and 27 types specific to Daggerheart reside in the core `storage/` package. Future systems would follow the same pattern, bloating the contract layer.

**Recommendation:** Move to `storage/systems/daggerheart/` sub-package. Each system gets its own contract namespace.

### F7.2: `ProjectionApplyTxStore` Type Assertion Coupling — Important

**Severity:** important

**Location:** `app/server_bootstrap.go` (projection apply callback)

`ProjectionApplyTxStore` contract defines only core stores, but concrete SQLite implementation also implements `DaggerheartStore`. Business logic uses type assertion to extract it — hidden coupling with no compiler guidance.

**Recommendation:** Either embed `DaggerheartStore` explicitly in the interface, or use an adapter/registry pattern for system stores.

### F7.3: `store_conversions.go` — Code Generation Opportunity

**Severity:** important

**Location:** `storage/sqlite/store_conversions.go` (933 lines)

Pure data mapping with highly repetitive patterns. Enum converters (switch on string, return constant) and row type adapters (3 identical functions for GetCampaignRow, ListCampaignsRow, ListAllCampaignsRow).

**Recommendation:** Implement code generation (`make generate` target) for enum and row adapters. Target: reduce from 933 to ~200 lines.

### F7.4: Reader/Writer Split — Excellent Consistency

**Severity:** style (no action needed)

Each domain consistently defines `*Reader` (read-only) and `*Store` (read+write) interfaces. Applied uniformly.

**Opportunity:** Encourage callers to depend on `*Reader` interfaces where mutations aren't needed.

### F7.5: `ErrNotFound` Sentinel — Excellent

**Severity:** style (no action needed)

Single sentinel in `storage/storage.go`. 139 `errors.Is()` checks across the codebase. Consistent and correct.

### F7.6: Transaction Boundary Documentation — Gap

**Severity:** minor

**Location:** `storage/sqlite/store_projection_outbox.go`

Exactly-once outbox pattern is correctly implemented (INSERT OR IGNORE checkpoint + transactional apply + rollback on failure). But transaction isolation, idempotency semantics, retry backoff, and dead-letter threshold are underdocumented.

**Recommendation:** Add detailed doc comments explaining idempotency checkpoint, isolation guarantees, and retry strategy.

### F7.7: Exactly-Once Failure Case Testing — Partial

**Severity:** minor

Core idempotency (duplicate and concurrent duplicate) well-tested. Missing: callback error rollback verification, dead-letter threshold test (8 attempts), processing lease timeout test.

**Recommendation:** Add failure-case tests.

### F7.8: SQLite Configuration — Correct

**Severity:** style (no action needed)

WAL mode, 5s busy timeout, foreign keys ON, synchronous NORMAL. All pragmas verified at startup.

### F7.9: Connection Pooling — Minor Gap

**Severity:** minor

No explicit `SetMaxOpenConns`/`SetMaxIdleConns` configuration. Relies on `database/sql` defaults and SQLITE_BUSY retry logic.

**Recommendation:** Add explicit connection pool configuration appropriate for SQLite's single-writer model.

## Cross-References

- **Phase 6** (Projection): Projection apply store contracts
- **Phase 9** (Module Extension): System-specific storage contracts
- **Phase 11** (Configuration): Storage bundle lifecycle
