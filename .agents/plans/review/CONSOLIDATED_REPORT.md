# Game Service Comprehensive Review — Consolidated Report

**Date:** 2026-03-22
**Scope:** 14 review passes across ~250 packages of the game service
**Individual pass reports:** `.agents/plans/review/pass01_*.md` through `pass14_*.md`

---

## Executive Summary

The game service is architecturally sound and well above average for a codebase of its size. The event-sourcing pipeline, storage layer, authorization matrix, and domain boundaries are all well-designed. The review found **no critical production bugs** but identified **~150 findings** across 14 passes, organized below by cross-cutting theme and priority.

The highest-impact opportunities are:
1. **Transport duplication** — ~70 packages repeat identical patterns; extractable with shared helpers
2. **Error handling convergence** — 4 coexisting error systems with hardcoded locale and silent swallowing
3. **Bootstrap boilerplate** — identical contract structs and registration functions across 6 aggregates
4. **Testing infrastructure** — triple-duplicated store fakes, no property-based tests
5. **Observability gaps** — audit read-method allowlist incomplete, severity mapping inflated

---

## Critical Findings (Correctness Risks)

These could produce incorrect behavior in edge cases or mask bugs:

| # | Pass | Finding | File | Impact |
|---|------|---------|------|--------|
| C1 | P2 | Error normalization divergence: same `storage.ErrNotFound` produces different gRPC codes in core vs system transports | `campaigntransport/`, `damagetransport/` | Client-visible error inconsistency |
| C2 | P3 | `HandleDomainError` silently discards unknown errors without logging | `platform/errors/grpc.go:33` | Debugging blind spot |
| C3 | P3 | Hardcoded `"en-US"` locale in 3 error conversion paths | `domainwrite/transport.go:140`, `grpcerror/helper.go:20` | i18n broken for non-English |
| C4 | P4 | `CharacterReadinessChecker` doc says optional but startup validator rejects modules without it | `module/registry.go:44` vs `registries_validation_system.go:41` | Misleading API contract |
| C5 | P6 | Soft nil checks on `SceneInteraction`/`SessionInteraction` bypass store precondition system | `projection/apply_scene.go:56,127` | Silent projection data loss |
| C6 | P7 | `validateEmptyPayload` uses `string(raw) != "{}"` — rejects `null`, whitespace-padded `{}` | `campaign/registry.go:306` | Valid requests rejected |
| C7 | P10 | `BatchAppendEvents` doesn't validate all events share same campaign ID | `eventjournal/store_events_append.go:166` | Silent event misattribution |
| C8 | P12 | `classifyMethodKind` missing 8+ read methods — audit logs report reads as writes | `interceptors/telemetry.go:178` | Misleading audit data |
| C9 | P1 | No startup validator checks `StateFactory` output is type-compatible with fold router | `module/registry.go`, `registries_validation_system.go` | Wiring bug surfaces only at first event |

---

## High-Priority Anti-Patterns

### Transport Layer Duplication (Pass 2)

The transport layer's highest-impact issue is structural duplication across ~70 packages:

- **`SystemCommandInput` struct** declared identically in 10 packages with inconsistent naming
- **`CampaignStore` interface** redeclared 11 times, `SessionGateStore` 12 times
- **Write-path ceremony** (actor→marshal→build→execute→reload) repeated at 27+ call sites
- **Campaign load + system guard + validate** preamble repeated in every system transport handler
- **`ApplyErrorWithDomainCodePreserve`** defined in 3 places with identical implementations

**Recommended fix:** Extract shared `SystemCommandInput` struct, promote `CampaignStore`/`SessionGateStore` to shared contracts, generalize `sessionCommandExecutor` pattern into a shared `EntityCommandExecutor`.

### Error Handling Architecture (Pass 3)

Four coexisting error systems create cognitive overhead:

- Plain sentinels (`errors.New`) for infrastructure errors → mapped to `codes.Internal` even when they represent validation failures
- Structured `apperrors.Error` with codes → properly mapped via `HandleError`
- Domain rejections (string codes) → i18n via catalog lookup with hardcoded `"en-US"` fallback
- gRPC statuses → final client-visible format

Key issues:
- User-facing validation errors (`ErrPayloadInvalid`, `ErrTypeUnknown`) use plain sentinels → `codes.Internal` instead of `codes.InvalidArgument`
- No validation that rejection codes have corresponding i18n catalog entries
- No system-level rejection code uniqueness validation (Daggerheart's 25+ codes unchecked)
- String literals used instead of `command.RejectionCode*` constants in 6+ locations

### Bootstrap Boilerplate (Pass 5)

Registration ceremony creates unnecessary contributor friction:

- Identical `commandContract`/`eventProjectionContract` structs independently declared in 5 packages
- 30+ nearly-identical package-level functions across 6 aggregates (5 per aggregate)
- `CoreDomains()` hardcoded list — forgetting an entry compiles but fails at startup
- `SystemDescriptor` duplicates `Module.ID()`/`Version()` fields
- 8 validation functions hard-call `CoreDomains()` as a global instead of accepting parameters

---

## Medium-Priority Findings by Theme

### Type Safety (Pass 1)
- Ad-hoc type switches in `ReplayGateStateLoader` duplicate `AssertState` logic (~30 lines)
- `aggregateState()` in `CoreDecider` silently returns zero-value on type mismatch
- `ReplayStateLoader.StateFactory` is `func() any` when all callers return `aggregate.State`

### Interface Design (Pass 4)
- `Applier` 19-field struct remains flat despite `StoreGroups` existing — half-completed migration
- 5 `registryBootstrap` shim methods exist as "historical test seams" with no removal criteria
- Three registries (`module.Registry`, `MetadataRegistry`, `AdapterRegistry`) duplicate identical key/registration code

### Projection System (Pass 6)
- Tautological bridge test `TestRegisteredHandlerTypes_MatchesProjectionHandledTypes` should be deleted
- `BuildExactlyOnceApply` omits `Auditor`, silencing gap audit events in outbox path
- `validatePreconditions` reports only first missing store vs `ValidateStorePreconditions` reports all

### Aggregate Patterns (Pass 7)
- Action decider hardcodes `"COMMAND_TYPE_UNSUPPORTED"` instead of using shared constant
- Action rejection codes lack domain prefix (e.g., `REQUEST_ID_REQUIRED` vs `CAMPAIGN_*`)
- Event definition `Intent` field inconsistently set (omitted in campaign/participant/character)
- `now` normalization placement varies across aggregates (dispatcher-level vs sub-function-level)
- Session decider does not guard `state.Started` for gate/spotlight/OOC/AI-turn commands

### Core ID Boundary (Pass 8)
- 3 Daggerheart-specific IDs (`AdversaryID`, `EnvironmentEntityID`, `CountdownID`) in core `domain/ids` package
- All 40 consumers are within Daggerheart-scoped packages — clean mechanical fix

### Daggerheart Exemplar (Pass 9)
- `internal/decider/exports.go` re-exports 34 constants for root-package testing — large maintenance tax
- `RegistrySystem` returns nil for `StateHandlerFactory` and `OutcomeApplier` — entire bridge hierarchy unimplemented
- Dual `StatePatch` type across adapter and projection packages
- `rest_package.go` is a 471-line monolith
- No second-system onboarding guide exists

### Storage (Pass 10)
- `EventStore` and `IntegrationOutboxStore` lack Reader splits
- `CampaignReadStores` composite name is misleading — embeds full Store interfaces including writes
- `SceneGate` uses `[]byte` while `SessionGate` uses `map[string]any` for same concept

### Testing (Pass 11)
- DaggerheartStore fakes duplicated in 3 separate packages (~1,300+ lines)
- Core store fakes (Campaign, Character, Session, Event) also duplicated across 2 packages
- `projection/testevent` defines parallel Event type with sync-drift risk
- No property-based or fuzz tests in an event-sourced system
- 224 Lua scenarios but only 52 in smoke manifest with no completeness contract
- `SequentialIDGenerator` produces garbage characters after 9 invocations

### Authorization (Pass 12)
- Inline role checks in 3 handler files bypass domain policy matrix
- `ValidateSessionLockPolicyCoverage` is namespace-level, not per-command
- Session lock interceptor is unary-only — no streaming equivalent
- `allActions`/`allResources` slices require manual sync with no compile-time enforcement

### Observability (Pass 13)
- `AuditInterceptor` creates new `Emitter` per request instead of reusing
- All non-OK errors logged at ERROR severity regardless of code (client errors inflated)
- Auth copy uses ~40 inline English fallbacks alongside catalog keys
- OTel uses `AlwaysSample()` with no production sampler knob

### Go Idioms (Pass 14)
- `Handler.Execute` — the primary exported method — lacks a doc comment
- Architecture doc `grpc-write-path.md` references 5 function names that no longer exist
- `log.Printf` used in 2 places while rest of codebase uses `slog`
- Unused `_ context.Context` on `evaluateGate`

---

## Positive Observations

The review confirmed many strong patterns:

- **Storage layer** is exemplary: clean Reader/Store split, append-only triggers, HMAC chain integrity, per-campaign HKDF key derivation, exponential backoff with dead-letter threshold
- **Module interface** (8 methods) is appropriately sized and cohesive for a game-system plugin contract
- **`fold.CoreFoldRouter[S]`** and `TypedDecider[S]` provide excellent type-safe wrappers over necessary `any` boundaries
- **Architecture contract tests** enforce doc.go presence, import isolation, and package conventions
- **Coverage floor ratchet** tracks 40+ packages with per-package floors from 57% to 100%
- **Aggregate fold signatures** and value-type state are 100% consistent across all 6 aggregates
- **Projection bitmask preconditions** provide O(1) startup validation and per-event safety
- **Event parity tests** cover both core and system events
- **Write-path architecture guard** uses AST scanning to enforce no-inline-apply invariant

---

## Recommended Action Priority

### Tier 1: Fix Now (correctness risks, highest ROI)
1. Fix hardcoded `"en-US"` locale — thread locale through context (C3)
2. Add logging before `HandleDomainError` discards unknown errors (C2)
3. Fix `classifyMethodKind` read allowlist or invert to default-read (C8)
4. Fix `CharacterReadinessChecker` doc vs validator inconsistency (C4)
5. Fix `validateEmptyPayload` to accept `null` and whitespace-padded `{}` (C6)
6. Add campaign ID validation in `BatchAppendEvents` (C7)
7. Unify error normalization between core-game and system transports (C1)

### Tier 2: Reduce Duplication (highest maintenance impact)
8. Extract shared `SystemCommandInput` struct for system transports
9. Promote `CampaignStore`/`SessionGateStore` to shared contracts
10. Extract shared `DomainRegistrar` type for aggregate registration boilerplate
11. Consolidate triple-duplicated store fakes into canonical implementations
12. Complete `Applier` `StoreGroups` embedding migration

### Tier 3: Strengthen Contracts (contributor safety)
13. Add startup validator for StateFactory-fold type compatibility (C9)
14. Replace string literals with `command.RejectionCode*` constants
15. Add rejection code i18n catalog coverage validation
16. Add system rejection code uniqueness validation
17. Move Daggerheart-specific IDs to `domain/systems/daggerheart/ids/`
18. Delete tautological `TestRegisteredHandlerTypes_MatchesProjectionHandledTypes`
19. Fix projection soft nil checks or add `storeSceneInteraction` to requirements

### Tier 4: Documentation and Polish
20. Update `grpc-write-path.md` stale function references
21. Add `Handler.Execute` doc comment
22. Create second-system onboarding guide
23. Add `TypedFolder`/`TypedDecider` mention to game-system authoring guide
24. Migrate `log.Printf` to `slog` in `profile_snapshot.go`
25. Remove unused `_ context.Context` from `evaluateGate`
26. Add audit severity mapping (client errors → WARN, server errors → ERROR)

---

## Metrics

| Metric | Value |
|--------|-------|
| Passes completed | 14 |
| Total findings | ~150 |
| Critical (correctness risk) | 9 |
| High (anti-pattern, major duplication) | ~25 |
| Medium (inconsistency, contributor friction) | ~60 |
| Low (polish, documentation) | ~40 |
| Positive observations | ~20 |
| Files read across all passes | ~500+ |
