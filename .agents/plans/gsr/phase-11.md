# GSR Phase 11: Configuration & Bootstrapping

## Summary

The configuration and bootstrapping layer is **excellent**. Explicit dependency injection through `serverBootstrapConfig`, LIFO rollback semantics, comprehensive startup phase documentation, and clear required vs. optional dependency semantics. No architectural concerns found.

## Findings

### F11.1: serverBootstrapConfig (14 Function Fields) — Sound

**Severity:** style (no action needed)

**Location:** `app/bootstrap.go:44-60`

14 function fields mapping to 8 startup phases plus supporting utilities. This is legitimate, testable dependency injection — not a god object. Each field maps to a phase-specific concern. Paired with `newServerBootstrapWithConfig()` constructor and `normalizeServerBootstrapConfig` for nil-defaults.

### F11.2: normalizeServerBootstrapConfig — Idiomatic

**Severity:** style (no action needed)

**Location:** `app/bootstrap.go:86-153`

Nil-field defaults are idiomatic Go. Functional options or builder patterns would add boilerplate not justified for ~14 fields. Tested via `TestNormalizeServerBootstrapConfigDefaults`.

### F11.3: Startup Phase Ordering — Well-Documented & Tested

**Severity:** style (no action needed)

**Location:** `app/bootstrap.go:299-434`, documented in `docs/running/game-startup-phases.md`

8 phases in correct dependency order: Registry → Network → Storage → Domain → Systems → Dependencies → Transport → Runtime. Each phase has documented failure semantics and rollback strategy.

### F11.4: startupRollback — Clean & Correct

**Severity:** style (no action needed)

**Location:** `app/startup_rollback.go:1-26`

LIFO stack with idempotent cleanup and explicit release. Tested for ordering (`TestStartupRollbackCleanup_ReverseOrder`) and release semantics (`TestStartupRollbackRelease_SkipsCleanup`).

### F11.5: Environment Loading — Pragmatic

**Severity:** style (no action needed)

**Location:** `app/server_config.go:32-76`

Lazy validation with sensible defaults. Database paths default to `data/` subdirectory, service addresses default to standard ports. Missing critical env vars caught downstream. Documented in `docs/running/configuration.md`.

### F11.6: Graceful Shutdown — Correct Semantics

**Severity:** style (no action needed)

**Location:** `app/server_runtime.go:36-62`, `app/server_runtime_serve.go:18-45`

Context-driven cancellation, `composeRuntimeStops` for reverse-order worker cleanup, `runGRPCServeLoop` with `GracefulStop`. Tested for context cancellation and serve error handling.

**Minor observation:** No explicit timeout on graceful shutdown — relies on caller using `context.WithTimeout()` on root context. By design.

### F11.7: disabledDomain Pattern — Clean

**Severity:** style (no action needed)

**Location:** `app/domain.go:18-45`

`FRACTURING_SPACE_GAME_DOMAIN_ENABLED=false` disables write path. Commands return `errDomainWritePathDisabled` (fail-closed, not silent no-op). Projection-only mode fully supported. Both paths tested.

### F11.8: gRPC Dial Timeouts — Centralized

**Severity:** style (no action needed)

**Location:** `app/server_bootstrap.go:144-232`

`timeouts.GRPCDial` (2s) centralized in platform package. Auth failure is fatal; Social/AI/Status degradation is graceful with logged warnings. Appropriate for same-DC deployment.

## Minor Opportunities (Not Issues)

1. Consider documenting `serverBootstrapConfig` pattern in `doc.go` as a project template
2. Consider integration test for full startup→shutdown cycle (individual phase tests are solid)

## Strengths

- Explicit dependency injection with testable seams
- LIFO rollback with reverse-order cleanup
- Clear required (Auth) vs. optional (Social, AI, Status) dependency semantics
- Comprehensive startup phase documentation
- Projection-only mode support via `disabledDomain`
- Environment configuration fully documented in `docs/running/configuration.md`

## Cross-References

- **Phase 7** (Storage): Storage bundle lifecycle managed during bootstrap
- **Phase 14** (Observability): Status runtime configured in startup phase 8
- **Phase 8** (gRPC Transport): gRPC server created and registered in startup phase 7
