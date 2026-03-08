# GSR Phase 12: Testing Infrastructure & Patterns

## Summary

Testing infrastructure is **solid with consistent patterns**. Test isolation via per-test databases, pure domain unit tests, production-grade fakes, and comprehensive scenario runner documentation. One caution: mega test files (3793 lines) are approaching cognitive limits.

## Findings

### F12.1: Test Isolation — Well-Implemented

**Severity:** style (no action needed)

Integration tests use separate SQLite databases per test via `t.TempDir()`. Optional shared fixture via `INTEGRATION_SHARED_FIXTURE` env var. Unit tests use in-memory fakes with no shared state.

### F12.2: `sync.Once` Shared Fixture — Correct

**Severity:** style (no action needed)

Used for keygen, content template caching, and optional server fixture. Resources are read-only or deterministic after initialization. Error handling via immediate `t.Fatalf()`.

### F12.3: Fake Implementations — Production-Grade

**Severity:** style (no action needed)

~10 fake types with error injection, constraint enforcement (user uniqueness), and call tracking. Pagination logic mirrors production behavior. All fakes implement only required interface methods.

### F12.4: Table-Driven Tests — Deliberately Mixed

**Severity:** style (no action needed)

Authorization logic uses table-driven (many similar cases). Domain deciders use individual tests (different business rules). Projection appliers use individual tests (one handler per test). Naming is consistent and intention-revealing.

### F12.5: Build Tag Separation — Clean

**Severity:** style (no action needed)

All integration tests use `//go:build integration`. Unit tests in default namespace. No tag leakage.

### F12.6: Error Path Coverage — Partial

**Severity:** minor

Happy path and null-pointer errors covered. Missing: partial failure during projection apply (constraint violations), event envelope cryptographic errors, transient network errors with retry.

**Recommendation:** Add scenarios simulating constraint violations during projection apply.

### F12.7: Domain Tests — Pure Unit

**Severity:** style (no action needed)

Deciders are pure functions (no storage, no clock dependency). Projection appliers use in-memory fakes (reasonable middle ground). No tests accidentally call real storage.

### F12.8: Scenario Runner Docs — Excellent

**Severity:** style (no action needed)

Canonical, current, extension path documented. Clear DSL examples, command matrix by audience, shard strategy with deterministic hash selection.

### F12.9: Test Naming — Consistent

**Severity:** style (no action needed)

Intention-first naming: `TestDecide<Command>_<Expectation>`, `TestApply<Event>_<Scenario>`. No drift detected.

### F12.10: Mega Test Files — Approaching Limits

**Severity:** minor

`applier_test.go` (3793 lines), `fakes_test.go` (1346 lines). Comment sections help navigation but files are large.

**Recommendation:** Consider extracting fakes into individual files per domain. Add TOC comments at top of large test files.

## Cross-References

- **Phase 6** (Projection): Applier test coverage
- **Phase 7** (Storage): Exactly-once failure testing
- **Phase 8** (gRPC Transport): Service test patterns
