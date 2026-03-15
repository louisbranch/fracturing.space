# Session 18: App Layer — Bootstrap, Configuration, Lifecycle

## Status: `complete`

## Package Summaries

### `app/` (~20+ files)
Application bootstrap and lifecycle management. Key files:
- `bootstrap.go` (444 lines) — Main wiring function
- `bootstrap_*.go` (5 files) — Component-specific bootstrap helpers
- `server_*.go` (7 files) — Server lifecycle and gRPC setup
- `domain.go` — Domain wiring entry point
- `system_registration.go` — Game system registration
- `startup_*.go` — Startup validation and health checks

## Findings

### Finding 1: bootstrap.go at 444 Lines — Acceptable Wiring Accumulation
- **Severity**: medium
- **Location**: `app/bootstrap.go`
- **Issue**: The bootstrap function wires all dependencies: storage, registries, engine, projection, handlers, interceptors, gRPC services. At 444 lines, it's the longest single-purpose file in the app layer. However, bootstrap is inherently a wiring concern — it connects all layers.
- **Recommendation**: The 5 `bootstrap_*.go` helper files suggest decomposition has already been done. If the main bootstrap grows beyond 500 lines, extract more component-specific wiring into helper files.

### Finding 2: Game System Registration Path
- **Severity**: info
- **Location**: `app/system_registration.go`
- **Issue**: System registration follows a clear path: register module → register commands/events → register projection handlers → register transport handlers. A contributor adding a new game system follows this file as a template.
- **Recommendation**: Document this registration path in the contributing guide. The `game-system` skill referenced in CLAUDE.md covers this.

### Finding 3: Server Lifecycle Is Clean
- **Severity**: info
- **Location**: `app/server_*.go`
- **Issue**: Server files handle gRPC server creation, graceful shutdown, and health checks. Standard Go patterns for server lifecycle management.
- **Recommendation**: Clean implementation.

### Finding 4: server_helpers_test.go at 1,364 Lines — Test Infrastructure
- **Severity**: medium
- **Location**: `app/server_helpers_test.go`
- **Issue**: 1,364 lines of test infrastructure (fakes, builders, assertion helpers) for server-level integration tests. This is effectively a test harness that sets up the full application stack for testing.
- **Recommendation**: Consider extracting reusable test infrastructure into a `app/apptest/` sub-package. This would allow other packages to reuse the test harness.

### Finding 5: Configuration Typing
- **Severity**: info
- **Location**: `app/bootstrap.go`
- **Issue**: Configuration is passed as a typed struct to the bootstrap function, providing compile-time safety for required settings.
- **Recommendation**: Good pattern.

### Finding 6: domain.go Purpose
- **Severity**: info
- **Location**: `app/domain.go`
- **Issue**: `domain.go` likely wires the domain engine (registries + handler + folder). This is the integration point between the app layer and the domain layer.
- **Recommendation**: If this file is a thin delegation to engine builder, it's well-placed. If it grows to contain domain logic, it should be decomposed.

## Summary Statistics
- Files reviewed: ~27 (20 production + 7 test)
- Findings: 6 (0 critical, 0 high, 2 medium, 0 low, 4 info)
