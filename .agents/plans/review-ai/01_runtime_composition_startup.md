# P01: Runtime Composition and Startup Boundary

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the AI runtime entrypoints and composition root to decide whether startup, config loading, dependency wiring, and process lifecycle are easy to understand, easy to test, and architecturally clean for contributors.

Primary scope:

- `cmd/ai`
- `internal/cmd/ai`
- `internal/services/ai/app`

## Progress

- [x] (2026-03-23 04:00Z) Reviewed `cmd/ai`, `internal/cmd/ai`, `internal/services/ai/app`, and their current tests.
- [x] (2026-03-23 04:00Z) Captured initial findings F01-F05 covering duplicated startup validation, monolithic assembly, registration drift risk, cross-service metadata helper placement, and missing composition-level contract tests.
- [x] (2026-03-23 04:08Z) Synthesized the target runtime/package shape and cutover order for startup-boundary cleanup.

## Surprises & Discoveries

- The package boundary is good at the top level: `app` is clearly the composition root and `cmd/ai` stays thin.
- The internal composition code is still concentrated heavily in `newServerWithRuntimeConfig` and `buildHandlers`, so the package boundary is cleaner than the function-level boundary.
- Optional game connectivity changes real runtime behavior, but current `app` tests mostly cover env parsing and lifecycle, not the optional-dependency matrix.

## Decision Log

- Decision: Keep `internal/services/ai/app` as the composition root package, but treat the current monolithic assembly functions as refactor targets rather than the final intended shape.
  Rationale: The package itself is correctly placed; the issue is the breadth of wiring concentrated into single functions and structs.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

P01 is stable enough to treat as complete for planning purposes. The composition-root package is the right package boundary, but it needs four structural cleanups before the startup story is contributor-friendly: one startup validation authority, smaller workflow assembly modules, table-driven service registration, and a shared internal-identity metadata helper outside game transport.

## Context and Orientation

Read before recording findings:

- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`
- `internal/services/ai/app/doc.go`
- `internal/services/ai/app/config.go`
- `internal/services/ai/app/server.go`
- `internal/cmd/ai/ai.go`
- `cmd/ai/main.go`

## Plan of Work

Inspect:

- composition-root size and responsibility spread
- config parsing/validation duplication or drift
- startup invariants and error quality
- service registration and wiring readability
- provider/runtime policy leakage into startup code
- internal service identity and cross-service client setup
- testability of runtime assembly

## Current Findings

### F01: Startup validation has two authorities for the encryption-key invariant

Category: anti-pattern, contributor friction

Evidence:

- `internal/services/ai/app/config.go:54-75` validates that `EncryptionKey` exists and decodes as base64.
- `internal/services/ai/app/server.go:87-107` repeats the same empty/decode checks before constructing the sealer.
- Tests cover both entrypaths independently in `config_test.go` and `server_test.go`.

Impact:

- Contributors have to remember whether config validation or server construction is the authoritative place for startup invariants.
- Adding a new invariant risks partial duplication between `runtimeConfig.Validate()` and `newServerWithRuntimeConfig()`.

Refactor direction:

- Make `runtimeConfig.Validate()` the single source of truth for startup invariants.
- Either store a validated/decoded encryption payload on `runtimeConfig`, or move sealer construction into a small explicit dependency-builder helper that assumes `runtimeConfig` is already valid.

### F02: `buildHandlers` is a monolithic workflow assembler with a broad dependency bag

Category: anti-pattern, maintainability risk

Evidence:

- `internal/services/ai/app/server.go:215-230` defines `handlerDeps` with store, config, multiple adapter maps, managers, loader, clients, and a managed connection.
- `internal/services/ai/app/server.go:245-415` builds every service and handler family inline.

Impact:

- A contributor changing one workflow family must read unrelated credential, agent, orchestration, campaign-debug, and provider-grant wiring.
- The function is the real runtime map, but it is not organized by coherent runtime modules.
- Refactoring one workflow’s dependencies risks accidental churn in the global dependency bag.

Refactor direction:

- Split assembly by bounded workflow module, for example: credential/provider-auth, agent/invocation/access, campaign/orchestration/debug.
- Replace `handlerDeps` with smaller config structs per assembly function or a typed runtime module struct.
- Return a registration list rather than a concrete bag of handler pointers when possible.

### F03: Service registration and health registration are duplicated manually

Category: missing best practice

Evidence:

- `internal/services/ai/app/server.go:417-438` registers each gRPC service once and then repeats the service names for health status registration.

Impact:

- Adding a new service requires updating two manual lists in the same function.
- A missing health registration would be easy to miss in review because the type system does not connect the two lists.

Refactor direction:

- Use one registration table containing the handler and health service name together.
- Drive both `Register*Server` and `SetServingStatus` from the same table.

### F04: AI startup depends on a game-transport metadata helper package

Category: architecture boundary leak

Evidence:

- `internal/services/ai/app/internal_identity.go:7` imports `internal/services/game/api/grpc/metadata`.

Impact:

- AI runtime composition depends on an implementation helper that lives under the game service transport tree.
- This weakens service isolation and makes contributor reasoning harder: AI’s internal-identity logic looks shared, but the shared contract is hidden under game-owned transport paths.

Refactor direction:

- Move the metadata helper to a shared package under `internal/services/shared` or `internal/platform`.
- Keep game and AI transport packages consuming the shared helper, not each other.

### F05: Composition-level tests do not cover the optional game-dependency matrix

Category: testability gap

Evidence:

- `internal/services/ai/app/server_test.go:25-239` covers env/config validation, serve lifecycle, and prompt-builder fallbacks.
- No test currently asserts which handlers or runtime capabilities degrade when `GameAddr` is empty or the optional managed connection is unavailable.

Impact:

- Optional runtime behavior exists, but its contract is implicit.
- Contributors can break degraded startup behavior or campaign-orchestration readiness without a focused composition-level test failing.

Refactor direction:

- Add one composition contract test around `newServerWithRuntimeConfig()` or a smaller extracted builder that covers:
  - no game dependency
  - game dependency present
  - partial prompt-instruction availability
- Keep the test at the composition seam rather than adding broad unit tests for thin command entrypoints.

## Concrete Steps

1. Map runtime entrypoints from `main` to `server.New/Run`.
2. Identify which dependencies are created here vs should move behind package seams.
3. Check whether `app` is purely wiring or also hides business policy.
4. Audit current tests to see if runtime behavior is exercised at the right seam.
5. Record specific refactor proposals with deletion candidates.
6. Convert findings into a target runtime shape with an ordered cutover plan.

## Target Runtime Shape

Keep the current package split:

- `cmd/ai`: process entrypoint only
- `internal/cmd/ai`: flag/env parsing and process startup only
- `internal/services/ai/app`: composition root only

Refactor `internal/services/ai/app` into these internal seams:

1. `runtimeConfig` remains the single startup-config authority.
   It validates invariants once and exposes small dependency-builder helpers for decoded key material, optional OAuth config, and runner settings.
2. `buildRuntimeDeps(cfg)` constructs shared runtime dependencies.
   It opens the store, builds the sealer, dials optional game clients, constructs provider adapters, and returns a typed runtime dependency struct.
3. `buildWorkflowModule*` helpers assemble bounded workflow slices.
   Suggested slices:
   - credentials/provider auth
   - agent/invocation/access
   - campaign orchestration/debug/artifacts/references
4. `registerServices` becomes table-driven.
   One service descriptor should hold the registration callback, health service name, and concrete server implementation.
5. internal service identity should consume a shared metadata helper package.
   AI startup and AI transport must not import helpers from `internal/services/game/api/grpc/...`.

## Cutover Order

1. Extract the shared service-ID metadata helper to a neutral shared package and switch AI startup/transport to it.
2. Make `runtimeConfig.Validate()` the single startup-invariant authority and remove duplicate encryption-key checks from server construction.
3. Introduce a typed runtime dependency builder so `newServerWithRuntimeConfig` stops constructing every dependency inline.
4. Split `buildHandlers` into workflow assembly helpers with smaller config structs.
5. Replace manual gRPC + health registration with one service registration table.
6. Add one composition-level contract test for the optional game dependency matrix and prompt-instruction degradation.

## Validation and Acceptance

- `go test ./internal/services/ai/app ./internal/cmd/ai`
- `go test ./internal/services/ai/...`

Acceptance:

- findings identify whether startup code is readable to a new contributor
- target runtime shape is explicit enough to implement without guesswork
- interface impact on env/config or server construction is recorded
- cutover order is explicit enough to execute without inventing new boundaries mid-refactor

## Idempotence and Recovery

- Runtime reading is side-effect free.
- If a later pass changes assumptions about composition ownership, update this file and the master plan together.

## Artifacts and Notes

- Record exact file references for any startup policy that should move out of `app`.
- Likely cutover sequence from current findings:
- move shared metadata helper out of game transport first
- centralize startup validation next
- split `buildHandlers` into workflow assembly helpers
- replace manual registration with one service table
- add one composition contract test for optional game connectivity

## Interfaces and Dependencies

Track any proposed changes to:

- runtime config shape
- `server.New` / `server.Run` construction boundaries
- command entrypoint contracts
