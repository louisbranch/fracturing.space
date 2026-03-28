# P02: gRPC Transport, Proto Surface, Auth Extraction, and Error Normalization

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the public AI gRPC surface and transport layer for duplication, inconsistent parsing/auth/pagination behavior, weak error mapping, and contributor friction when adding or changing RPCs.

Primary scope:

- `api/proto/ai/v1/service.proto`
- `internal/services/ai/api/grpc/ai`

## Progress

- [x] (2026-03-23 04:08Z) Reviewed handler roots, representative handler methods, shared helpers, campaign-context validation, and transport test helper layout.
- [x] (2026-03-23 04:08Z) Verified transport baseline with `go test ./internal/services/ai/api/grpc/ai`.
- [x] (2026-03-23 04:13Z) Synthesized the target transport helper/policy surface and cutover order.

## Surprises & Discoveries

- The package has a good high-level organization by workflow family, but handler method bodies still repeat the same unary transport scaffold heavily.
- Page-size clamping is centralized, while user-ID extraction and missing-user failure behavior are repeated by hand in nearly every user-scoped RPC.
- Campaign-scoped transport has a reusable validator, but that validator currently depends on a game transport metadata helper rather than a neutral shared contract.

## Decision Log

- Decision: Keep workflow-family handler grouping, but do not preserve the current one-root-file-plus-one-method-file shape if a smaller number of clearer files would reduce navigation cost.
  Rationale: The current grouping is conceptually correct, but several root files add very little beyond constructor validation and unimplemented-server embedding.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

P02 is stable enough to treat as complete for planning purposes. The transport layer has the right top-level workflow grouping, but it needs a clearer internal policy surface: one user-scoped unary helper path, one campaign-context validator path, and one explicit non-service error fallback. The current shape is workable, but the helper surface is too small relative to the amount of repeated handler scaffolding.

## Context and Orientation

Read before recording findings:

- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`
- `api/proto/ai/v1/service.proto`
- `internal/services/ai/api/grpc/ai/doc.go`
- `internal/services/ai/api/grpc/ai/transport_common.go`
- `internal/services/ai/api/grpc/ai/proto_helpers.go`
- `internal/services/ai/api/grpc/ai/service_errors.go`

## Plan of Work

Inspect:

- request validation consistency
- user/service identity extraction
- pagination defaults and token handling
- proto-to-domain mapping duplication
- root handler composition and family boundaries
- error/status normalization consistency
- handler test ergonomics vs helper complexity

## Current Findings

### F01: User-scoped unary RPCs repeat the same auth and request-validation scaffold

Category: anti-pattern, contributor friction

Evidence:

- The same `in == nil`, `userID := userIDFromContext(ctx)`, and `"missing user identity"` checks appear throughout:
  - `credentials_handlers.go`
  - `agents_handlers.go`
  - `invocation_handlers.go`
  - `provider_grants_handlers.go`
  - `access_requests_handlers.go`

Impact:

- Small transport changes require editing many handlers by hand.
- Shared user-auth semantics are implicit rather than encoded in one helper.
- Drift risk is low today because the pattern is simple, but it will rise as more RPCs are added.

Refactor direction:

- Introduce small transport helpers for:
  - required unary request presence
  - required caller user identity
  - common page extraction
- Keep request-shape validation near each RPC, but centralize the generic transport scaffold.

### F02: Handler-family root files add file-count overhead with very little unique value

Category: contributor friction

Evidence:

- Files such as `agent_handlers_root.go`, `invocation_handlers_root.go`, `campaign_orchestration_handlers_root.go`, and `access_request_handlers_root.go` mostly contain:
  - the handler struct
  - a tiny config struct
  - one `New*Handlers` constructor

Impact:

- Contributors often have to bounce between a root file and the actual method file just to understand one small handler family.
- The split adds indirection without creating a stronger architectural seam.

Refactor direction:

- Collapse trivial root files into their main handler file where the family only has one dependency and no special shared state.
- Keep separate root files only when the family has meaningful extra state, such as campaign-context validation.

### F03: Transport error normalization has three dialects instead of one explicit policy

Category: missing best practice

Evidence:

- Most workflow handlers use `serviceErrorToStatus`.
- Campaign orchestration adds `campaignTurnGRPCError()` with app-error and context handling in `campaign_orchestration_handlers.go`.
- Campaign artifacts and system reference handlers wrap non-service failures directly with `status.Errorf(codes.Internal, ...)`.
- Campaign debug handlers branch between `serviceErrorToStatus` and inline `status.Errorf`.

Impact:

- A contributor cannot learn one transport error policy and apply it everywhere.
- Non-service dependency errors are handled ad hoc rather than through one transport-owned mapper.
- Error wording and code choices are more likely to drift as new handler families appear.

Refactor direction:

- Keep `serviceErrorToStatus` for service-layer errors.
- Add one transport-owned fallback mapper for non-service errors, with campaign-orchestration app-error handling as a specialized branch inside the same policy surface rather than a separate dialect.

### F04: Campaign-context validation is a good seam, but it still leaks cross-service metadata coupling

Category: architecture boundary leak

Evidence:

- `campaign_context_helpers.go` centralizes campaign authorization for debug/artifact RPCs.
- It imports `internal/services/game/api/grpc/metadata` to identify allowed internal callers.

Impact:

- Transport reuse for campaign-scoped handlers is stronger than the user-scoped path, but the seam still depends on a game-owned transport helper.
- The same boundary leak found in P01 exists inside the transport package too.

Refactor direction:

- Keep `campaignContextValidator` as the transport seam for campaign-scoped authorization.
- Move service-ID metadata extraction into a shared package and update both startup and transport code to consume the shared helper.

### F05: Transport tests rely on a large local harness that reconstructs service graphs family by family

Category: anti-pattern, testability risk

Evidence:

- `transport_test_helpers_test.go` contains extensive family-specific helper builders and local fake adapters.
- Several helpers rediscover the same store-casting and service-construction logic separately for agent, invocation, and orchestration handlers.

Impact:

- The transport package is paying for handler-construction complexity in its tests.
- Contributors need to learn a local harness mini-framework before adding or adjusting handler coverage.
- This increases the chance that transport tests become the easiest place to test service behavior that belongs lower in the stack.

Refactor direction:

- Shrink transport-local harness code by introducing a smaller set of reusable AI transport test builders, or by leaning more on package-local service tests for workflow behavior.
- Keep handler tests focused on request parsing, auth, and response/status mapping.

## Concrete Steps

1. Map all services and RPC families from the proto file to handler roots.
2. Identify repeated request/auth/pagination/error logic.
3. Check whether business logic leaks into handlers.
4. Compare transport helper reuse to the amount of local test harness code.
5. Propose a target transport seam with explicit responsibilities.
6. Convert findings into a target helper surface and cutover order.

## Target Transport Shape

Keep the current package boundary:

- `internal/services/ai/api/grpc/ai` remains the transport package
- handlers stay grouped by workflow family

Refactor the internal transport surface into these explicit policies:

1. One small user-scoped unary helper path.
   It should centralize:
   - required request presence
   - required caller user identity
   - common page-size extraction
   It should not hide RPC-specific input parsing.
2. One campaign-context helper path.
   `campaignContextValidator` is the right shape for campaign-scoped authz and should remain the campaign-only transport helper seam.
3. One transport-owned error fallback policy.
   - `serviceErrorToStatus` remains for service-layer errors.
   - a new fallback mapper handles non-service errors consistently.
   - campaign-orchestration app-error handling becomes a specialization of that fallback policy rather than a separate transport dialect.
4. Fewer low-signal root files.
   Handler root files should exist only when the family has meaningful shared state or constructor complexity.
5. Smaller transport test builders.
   Handler tests should use a compact builder surface focused on auth, parsing, and mapping, not reconstruct service graphs repeatedly.

## Cutover Order

1. Move service-ID metadata extraction to a shared package and update transport campaign-context helpers to use it.
2. Add a small user-scoped unary helper for request/user/page extraction and convert the simplest handler families first.
3. Introduce one explicit non-service fallback error mapper and migrate artifact, system-reference, debug, and orchestration handlers to it.
4. Collapse trivial root files into their main handler files where no extra shared state exists.
5. Shrink `transport_test_helpers_test.go` around the new helper surface and keep handler tests transport-specific.

## Validation and Acceptance

- `go test ./internal/services/ai/api/grpc/ai`
- `go test ./internal/services/ai/...`

Acceptance:

- target transport helper surface is explicit
- public/proto interface risks are recorded
- contributor workflow for adding a new RPC is either validated or marked for cleanup
- transport findings distinguish healthy explicitness from drift-prone duplication
- cutover order is explicit enough to implement without re-deciding transport policy

## Idempotence and Recovery

- Treat proto contract changes as breaking by default unless there is a strong reason not to.

## Artifacts and Notes

- Record exact duplicated handler patterns that should be consolidated or deleted.

## Interfaces and Dependencies

Track any proposed changes to:

- RPCs, messages, enums, and pagination contracts in `service.proto`
- transport helper APIs
- error/status mapping policy
