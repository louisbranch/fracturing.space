# GSR Summary: Prioritized Action Backlog

## Overall Assessment

The game service architecture is **production-ready and well-engineered**. The codebase demonstrates strong architectural discipline with clean layer separation, no import cycles, pure domain logic, comprehensive startup validation, and consistent patterns. The review surfaced **no critical runtime bugs** — findings are primarily type-safety improvements, documentation gaps, and observability hardening.

## Severity Distribution

| Severity | Count | Areas |
|----------|-------|-------|
| Critical | 2 | Domain model type safety (F2.1, F2.3) |
| Important | 10 | Enum consistency, invariant enforcement, event payloads, storage coupling, observability |
| Minor | 15 | Documentation, conventions, decomposition opportunities |
| Style | 40+ | Patterns already sound — no action needed |

## Prioritized Action Backlog

### Priority 1: Critical — Domain Model Type Safety

These findings affect compile-time safety and long-term maintainability.

| ID | Finding | Phase | Effort | Status |
|----|---------|-------|--------|--------|
| F2.1 | **Introduce ID newtypes** for all identifier fields (CampaignID, ParticipantID, etc.) | 2 | High | **done** |
| F2.3 | **Address `map[module.Key]any` type safety** — make `AssertState[T]` fail-loud on nil | 2 | Medium | **done** |
| F2.2 | **Enforce typed enum constants** in all State structs (Role, Kind, SpotlightType, etc.) | 2 | Medium | **done** |

### Priority 2: Important — Invariant Enforcement & Event Design

| ID | Finding | Phase | Effort | Status |
|----|---------|-------|--------|--------|
| F2.4 | Add nil-guard constructors for aggregate maps | 2 | Medium | **done** |
| F2.5 | Add invariant-enforcing constructors for state types | 2 | Medium | deferred (low ROI vs churn) |
| F3.1 | Separate computed data from event payloads (before/after fields) | 3 | High | **done** |
| F3.2 | Resolve `extractSceneID` addressing inconsistency (gate events) | 3 | Medium | **done** (already documented in code) |
| F5.2 | Remove test-only `Handle()` API, consolidate to `Execute()` | 5 | Low | **done** |
| F5.4 | Remove runtime nil checks for startup-validated Handler fields | 5 | Medium | deferred (15+ test changes, no behavioral benefit) |

### Priority 3: Important — Storage & Transport Coupling

| ID | Finding | Phase | Effort | Status |
|----|---------|-------|--------|--------|
| F7.1 | Move `storage_daggerheart.go` to sub-package | 7 | Medium | deferred (71 file blast radius) |
| F7.2 | Eliminate `ProjectionApplyTxStore` type assertion coupling | 7 | Medium | **done** |
| F7.3 | Simplify `store_conversions.go` with generic JSON helper | 7 | Medium | **done** |
| F8.7 | Add mapper unit tests for proto↔domain conversions | 8 | Low | skipped (coverage already comprehensive in helpers_test.go) |
| F10.1 | Add rejection codes (~91) to i18n catalogs | 10 | High | **done** |

### Priority 4: Important — Observability Hardening

| ID | Finding | Phase | Effort | Status |
|----|---------|-------|--------|--------|
| F14.1 | Add streaming audit coverage | 14 | Medium | **done** |
| F14.2 | Promote domain rejections to audit events | 14 | Low | **done** |
| F14.3 | Emit audit events for projection gap/dead-letter detection | 14 | Low | **done** (gap detection; dead-letter deferred — storage layer) |

### Priority 5: Minor — Documentation & Cleanup

| ID | Finding | Phase | Effort | Status |
|----|---------|-------|--------|--------|
| F5.1 | Remove dead `HandlerConfig` alias | 5 | Trivial | **done** |
| F4.1 | Document DecideFunc variant adoption criteria | 4 | Low | **done** |
| F5.7 | Document gate evaluation ordering rationale | 5 | Trivial | **done** |
| F6.1 | Consider Applier decomposition (18 store fields) | 6 | High | closed (handlers already domain-grouped in 7 files; shared txStore and Campaign denorm writes make sub-appliers net-negative) |
| F6.2 | Replace `BuildErr` struct field with constructor error | 6 | Low | skipped (50 callers, working pattern) |
| F6.5 | Document watermark concurrency model | 6 | Trivial | **done** |
| F6.6 | Document handler ordering assumptions | 6 | Trivial | **done** |
| F7.6 | Document transaction boundary semantics | 7 | Low | **done** |
| F13.1 | Add "Core Design Philosophy" guide | 13 | Low | **done** |
| F13.6 | Add `platform/doc.go` categorization | 13 | Trivial | skipped (namespace dirs) |
| F15.1 | Add 3 missing root doc.go files | 15 | Trivial | skipped (namespace dirs) |
| F15.6 | Add visual diagrams (system registration, startup phases) | 15 | Medium | **done** |
| F17.3 | Centralize MCP gRPC error translation | 17 | Low | **done** |
| F3.6 | Document event type naming convention | 3 | Trivial | **done** |
| F3.7 | Document RegisterAlias usage criteria | 3 | Trivial | **done** |
| F4.6 | Document sessionStartRoute cross-domain exception | 4 | Trivial | **done** |
| F4.7 | Document entity ID resolution pattern | 4 | Trivial | **done** |
| F9.5 | Document StateFactory typed recovery pattern | 9 | Trivial | **done** |

## Phase Verdicts

| Phase | Area | Verdict | Key Strengths |
|-------|------|---------|---------------|
| 1 | Package Structure | **PASS** | Clean 7-layer stack, no cycles, excellent naming |
| 2 | Domain Model | **NEEDS WORK** | Type safety gaps, missing constructors |
| 3 | Event System | **GOOD** | Pure folds, complete validation; computed data concern |
| 4 | Command/Decision | **PASS** | Pure deciders, consistent rejection codes |
| 5 | Engine Orchestration | **GOOD** | Sound design; dead code and nil check cleanup needed |
| 6 | Projection Layer | **PASS** | Event parity validated, deterministic replay |
| 7 | Storage Contracts | **GOOD** | Clean contracts; coupling and codegen opportunities |
| 8 | gRPC Transport | **PASS** | Excellent organization, consistent patterns |
| 9 | Module Extension | **PASS** | Mature, well-validated, low contributor friction |
| 10 | Error Handling | **GOOD** | Clean boundaries; rejection codes need i18n |
| 11 | Configuration | **PASS** | Excellent DI, rollback, and shutdown |
| 12 | Testing | **PASS** | Strong isolation, consistent patterns |
| 13 | Core Utilities | **PASS** | Injectable, no inverted deps |
| 14 | Observability | **NEEDS WORK** | Audit gaps, no metrics, unstructured logging |
| 15 | Documentation | **GOOD** | Comprehensive; minor gaps in diagrams |
| 16 | Web/Admin | **PASS** | Clean separation, enforced module boundaries |
| 17 | MCP Transport | **PASS** | Complete gRPC parity, zero logic leakage |

## Execution Recommendations

1. **Start with Priority 1** (ID newtypes, enum enforcement) — foundational, enables safer refactoring everywhere
2. **Priority 2** can be tackled incrementally per domain package
3. **Priority 3** storage coupling fixes should precede adding new game systems
4. **Priority 4** observability work can be done independently
5. **Priority 5** minor items can be addressed opportunistically

## Architecture Strengths (Preserve These)

- Pure domain logic: deciders and folds are deterministic, replay-friendly
- 20+ startup validation steps catch misconfigurations before serving traffic
- Event parity tests prevent silent handler gaps
- Clean transport boundaries: gRPC, web, MCP all delegate correctly
- Explicit DI with testable seams throughout
- Comprehensive startup rollback with LIFO cleanup
- Auto-generated event catalogs stay in sync with source
- Architecture guardrail tests enforce module boundaries
