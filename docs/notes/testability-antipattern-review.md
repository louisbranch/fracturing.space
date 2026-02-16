---
title: "Testability Anti-Pattern Review"
parent: "Notes"
nav_order: 3
---

# Testability Anti-Pattern Review

This review is a snapshot on `2026-02-16` (branch `testability`) and is organized by divide-and-conquer clusters.

## Scope and Command Evidence

- `make cover`
- `go tool cover -func=coverage.out`
- `go tool cover -func=seed-cover.out`
- `go tool cover -func=mcp-service-cover.out`
- `go tool cover -func=web-cover.out`
- `go tool cover -func=game-app-cover.out`
- `go tool cover -func=admin-cover.out`
- `go tool cover -func=auth-cover.out`
- `go tool cover -func=chat-cover.out`
- `go tool cover -func=ai-cover.out`
- `go tool cover -func=maintenance-cover.out`
- `go tool cover -func=importer-cover.out`
- `rg -n "context\\.Background\\(|time\\.After\\(|time\\.NewTicker\\(|go run|sync\\.WaitGroup|SIGKILL|time\\.NewTimer|time\\.Sleep" <target files>`

- Baseline from this pass: `coverage.out` total `67.8%` statements.

## Method

- Split by clusters to minimize cross-coupled decisions.
- Keep cluster findings independent and prioritized by testability impact.
- Separate “coverage low because not exercised” from “coverage high but contract behavior is hidden.”

## Findings by Cluster

### Cluster 1 — Seed Process Orchestration (P0)

Target files:
- `internal/tools/seed/client.go`
- `internal/tools/seed/runner.go`

Anti-patterns:
- External process startup is hard to control:
  - `StartMCPClient` launches `go run ./cmd/mcp` directly (`client.go:27`).
- Process shutdown is hard to test without timing/fake OS helpers:
  - `Close` uses `time.After(5*time.Second)`, `SIGINT` + `SIGKILL` (`client.go:55-73`).
- Asynchronous line reader branch and goroutine join are environment-coupled (`client.go:120-148`).
- `executeStep` uses hard-coded `30*time.Second` timeout (`runner.go:147`).

Coverage signal:
- `client.go:27` `StartMCPClient` `0.0%`
- `client.go:55` `Close` `0.0%`
- `client.go:77` `WriteMessage` `0.0%`
- `runner.go:78` `ListScenarios` `0.0%`
- `runner.go:118` `executeStep` `52.7%`
- `runner.go:212` `createSeedUser` `0.0%`

Why this blocks testing:
- `StartMCPClient` and `Close` are bound to OS process behavior and timing semantics.
- Failure branches (command launch, timeout, kill paths) are not deterministically injected.
- Branches are currently effectively unverified by tests even when parent function has partial coverage.

Recommended seams:
- Inject command creation and process abstraction for lifecycle paths.
- Inject timeout strategy and command completion/failure behaviors.
- Add branch-specific tests for start failure, timeout, and createSeedUser transport failures.

### Cluster 2 — MCP HTTP Transport (P0)

Target files:
- `internal/services/mcp/service/http_transport.go`

Anti-patterns:
- Fixed timing defaults in package vars and in-path selectors:
  - `defaultRequestTimeout`, `defaultShutdownTimeout`, cleanup and heartbeat intervals.
- Test-only context ownership and cancellation coupling:
  - `NewHTTPTransport` initializes with `context.Background()`, and readiness cancellation/timeouts are real-time (`http_transport.go:109-129`, `117-119`, `226-227`, `481`, `571`, `1055`).
- Hard loops with real time (`cleanupSessions`, SSE heartbeat, request timeout branch).

Coverage signal:
- `cleanupSessions` `35.7%`
- `handleMessages` `69.5%`
- `handleSSE` `70.2%`
- `ensureServerRunning` `83.3%` with untestable readiness timing at branch level
- `readyAfterOrDefault` `66.7%`, `serverReadyTimeoutOrDefault` `66.7%`

Why this blocks testing:
- Timeout windows and tickers are not injectible, so negative/edge branches are time-dependent and flaky to force.
- Readiness and heartbeat behavior is difficult to force deterministically without fake clocks.

Recommended seams:
- Inject ready timer and timeout helper functions.
- Inject clock/ticker providers in transport tests (or explicit interval hooks).
- Add tests for validate host/origin reject matrix + request timeout + session readiness fallthrough.

### Cluster 3 — MCP Server Lifecycle (P1)

Target files:
- `internal/services/mcp/service/server.go`

Anti-patterns:
- Nil context is silently replaced with `context.Background()` in `resourceNotifier` and `serveWithTransport`.
- Fixed monitor loop/ticker timing and short `context.WithTimeout(...,5*time.Second)` durations.

Coverage signal:
- `monitorHealth` `31.2%`
- `serveWithTransport` `80.0%`
- `dialGameGRPC` `84.6%`

Why this blocks testing:
- Contract violations around nil context are not surfaced as explicit behavior.
- Deterministic control of health-check cadence/backoff is limited without injectable time.

Recommended seams:
- Make nil context behavior explicit and document contract at entry boundary.
- Inject health timer and timeout policy for test control.

### Cluster 4 — Service Startup Context and Shutdown Contracts (P1)

Target files:
- `internal/services/web/server.go`
- `internal/services/admin/server.go`
- `internal/services/auth/app/server.go`
- `internal/services/ai/app/server.go`
- `internal/services/game/app/server.go`
- `internal/services/chat/app/server.go`

Anti-patterns:
- Context ownership is split across constructors/listeners:
  - `web.NewServer` delegates to background context (`server.go:230`).
  - `chat.NewServer` delegates similarly (`chat/server.go:953`).
  - `auth.Serve`, `admin.ListenAndServe`, `ai.Serve`, `game.Serve` normalize nil context.
- Hardcoded cancellation/shutdown values:
  - fixed shutdown timeouts in auth/admin/others and retry timing loops.
- Retry loops with fixed backoff and non-injectable timers in admin.

Coverage signal:
- `admin.ListenAndServe` `72.2%`
- `auth.Serve` `0.0%` (package indicates largely untested)
- `ai.Serve` `0.0%`
- `chat.ListenAndServe` `25.5%`
- `game.Serve` `95.5%` (but contract path hidden)

Why this blocks testing:
- Nil context handling and cancellation behavior are not contractually tested.
- Time-sensitive branches in shutdown/retry loops are hard to control and can make negative-path tests brittle.

Recommended seams:
- Standardize and document context contract policy.
- Inject shutdown durations, retry delays, and cancellation paths.
- Add explicit nil/canceled context test matrix for each service boundary.

### Cluster 5 — Shared Context and gRPC Helper Utilities (P2)

Target files:
- `internal/tools/cli/context.go`
- `internal/platform/requestctx/user.go`
- `internal/platform/grpc/dial.go`
- `internal/platform/grpc/health.go`
- `internal/services/shared/grpcauthctx/grpcauthctx.go`
- `internal/services/game/api/grpc/metadata/metadata.go`

Anti-patterns:
- Nil context is normalized into `context.Background()` in helper utilities.

Coverage signal:
- These utilities are often not directly reflected as low coverage, so they can hide contract failures behind successful defaults.

Why this blocks testing:
- Existing tests may pass while context-ownership invariants are not validated.
- This pattern compounds with service-level behavior because callers rely on implicit defaults.

Recommended seams:
- Decide one explicit contract (fail-fast vs normalize) and apply consistently.
- Add focused contract tests for helper inputs and invalid usage.

### Cluster 6 — Importer Execution Path (P2)

Target files:
- `internal/tools/importer/content/daggerheart/v1/main.go`

Anti-patterns:
- `Run` defaults nil context to `context.Background()` (`main.go:61-63`).
- Store open path is directly coupled to concrete sqlite opener (`Run` path around `storagesqlite.OpenContent`, `main.go:91-97`).
- Iteration and import pipeline are not covered at `Run` level, masking error-path reachability.

Coverage signal:
- `main.go:61` `Run` `0.0%`
- package total `68.9%`

Why this blocks testing:
- End-to-end importer behavior cannot inject failure modes for missing directories, invalid payload files, or DB open failures.
- Context/FS/DB effects are entangled in one function.

Recommended seams:
- Extract IO/store open and context creation dependencies for deterministic failures.
- Add table tests for no-locale, invalid base-locale, empty payload, and DB open failure branches.

### Cluster 7 — Maintenance Store/Integrity Workflow (P2)

Target files:
- `internal/tools/maintenance/maintenance.go`

Anti-patterns:
- `openStores`/`openProjectionStore` are entirely untested (`0.0%` on both).
- `openEventStore` normalizes nil context to `context.Background()` before `VerifyEventIntegrity` (`maintenance.go:777-783`).
- Integrity workflow creates temp stores and uses real storage interactions without injection, reducing deterministic failure coverage.
- `checkSnapshotIntegrity` and several deep replay paths remain low and branch-sensitive (`882` onwards).

Coverage signal:
- `openStores` `0.0%`
- `openProjectionStore` `0.0%`
- `checkSnapshotIntegrity` `27.3%`
- package total `78.3%`

Why this blocks testing:
- Critical reliability error states (open/store verify/failure cleanup) are not directly asserted.
- Hard to inject OS/store failures at the seam points that matter for maintenance operators.

Recommended seams:
- Inject key interfaces for store creation/open and integrity verification execution.
- Add tests for path validation, open failure rollback, and temp-store lifecycle.

### Cluster 8 — Coverage Blind Spots and Trap (P2)

Observations:
- `internal/services/chat` package currently reports `total: 0.0%` coverage, which is likely not a behavior signal; it indicates under-tested service entry points.
- Several files now have moderate or high function-level coverage while still exposing hidden branches that are impossible to hit without additional seams (`NewServer` + hidden context/default branches).

Recommended follow-up:
- Treat blind-spot packages as P1 for coverage reconstruction after anti-pattern remediation.
- For each high-level constructor/listener, split behavior contracts from wiring.

### Cluster 9 — Scenario DSL and CLI Tooling (P2)

Target files:
- `internal/tools/scenario/{runner,runner_steps,runner_helpers,dsl}`
- `internal/tools/icondocgen/main.go`

Anti-patterns:
- `go test ./internal/tools/scenario` previously reported `54.0%` total coverage and many `0.0%` functions in `dsl.go`, `runner_helpers.go`, and `runner_steps.go`.
- `internal/tools/scenario/dsl.go:74` short-circuits validation with an unconditional `return nil`, so `validateScenarioComments` never executes its file-read/validation body.
- `internal/tools/scenario/dsl.go` and related files couple behavior directly to raw Lua state registration, making malformed script and callback failures hard to unit-simulate.
- `internal/tools/icondocgen/main.go` coverage is `0.0%`; `fatal` exits via `os.Exit(1)`, so function-level execution is not reachable in unit tests.

Status:
- Applied in this pass:
  - removed unconditional return in `validateScenarioComments`,
  - added regression tests for missing comment validation and `icondocgen` argument/error behavior,
  - extracted `run` and `writeOutput` seams in `internal/tools/icondocgen/main.go`.
- Cluster 9 follow-up coverage:
  - `internal/tools/scenario` now `55.0%` (up from `54.0%`),
  - `internal/tools/icondocgen` now `81.4%` (from `0.0%`).

Why this blocks testing:
- The unconditional comment validation branch is now covered, but many DSL method handlers remain untested.
- `internal/tools/scenario/dsl.go` still has branches where malformed script callbacks depend on integration-level behavior through `go-lua`.
- `icondocgen` still lacks deterministic tests for complete render/content contract (only generation lifecycle paths are currently covered).

Recommended seams:
- Add parser-level property/fuzz coverage for malformed Lua script and callback edge behavior.
- Introduce additional injection seams for parser/runner dependencies (e.g., parser factory, gRPC dial/clients) to drive deterministic failure paths.
- Add content-assertion tests for `icondocgen` catalog generation under controlled icon datasets.

## Existing Coverage Gaps to Address First

- Seed orchestration `StartMCPClient`, `Close`, `ListScenarios`, `createSeedUser`.
- MCP transport timeout/heartbeat/readiness branches and request validation matrices.
- MCP `monitorHealth` timing loops and `serveWithTransport` context contracts.
- Service entrypoint nil-context and shutdown/retry timing branches across web/admin/auth/ai/game/chat.
- Importer `Run` and maintenance `open*`/integrity failure paths.
- Chat package has no executed coverage signal; needs baseline coverage scaffolding before deeper branch audit.

## Validation Plan

- Validate each cluster independently with focused profile commands:
  - `go tool cover -func=seed-cover.out`
  - `go tool cover -func=mcp-service-cover.out`
  - `go tool cover -func=web-cover.out`
  - `go tool cover -func=game-app-cover.out`
  - `go tool cover -func=admin-cover.out`
  - `go tool cover -func=auth-cover.out`
  - `go tool cover -func=ai-cover.out`
  - `go tool cover -func=maintenance-cover.out`
  - `go tool cover -func=importer-cover.out`
- Maintain a per-cluster acceptance rule: branch with previously untested anti-pattern should become directly testable once seams are added in a follow-up implementation.

## Notes

- This document is findings-first and now includes one focused production remediation (cluster 9), documented as implemented.
- Baseline values may drift as local changes alter command outputs.
