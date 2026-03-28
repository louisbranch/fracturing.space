# P12: Cross-Service Seams, Operational Readiness, and Consolidated Refactor Roadmap

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the direct AI-owned seams into game and worker plus the AI service's operational readiness hooks, then collapse all pass findings into one ranked refactor roadmap with clear dependency order.

Primary scope:

- AI-owned game client usage in `internal/services/ai/app` and `internal/services/ai/service`
- session-grant and internal-identity validation paths
- AI integration tests that prove cross-service behavior
- master roadmap synthesis from P01-P11

## Progress

- [x] (2026-03-23 06:08Z) Reviewed direct cross-service seams, internal identity helpers, session-grant validation flow, debug-update ownership, and AI integration coverage inventory.
- [x] (2026-03-23 06:16Z) Built the final ranked roadmap and synced the master plan.
- [x] (2026-03-23 06:16Z) Validation complete for this documentation pass: `go test ./internal/services/ai/...` passed.

## Surprises & Discoveries

- The session-grant contract itself is healthy and well-documented; the main cross-service debt is not the grant format but the helper placement and raw collaborator usage around it.
- AI still imports game-owned gRPC metadata helpers in three different places: startup/internal identity, transport campaign-context validation, and gametool outgoing context assembly.
- Optional game connectivity is handled as composition-time nil capability, which means some behavior degrades silently while other flows fail later with precondition errors.
- The AI-owned debug-trace persistence path is cleaner than expected, but live update fanout is only an in-process broker today.
- Integration coverage is better than the earlier plan commands implied, but the canonical command surface was stale: AI integration tests require `-tags=integration`, and live-capture tests additionally require `-tags=integration,liveai`.

## Decision Log

- Decision: Extract shared internal gRPC metadata and service-identity helpers into a platform/shared package instead of continuing to import them from the game transport tree.
  Rationale: P01, P02, and P12 all found the same ownership leak from different directions. The clean fix is one shared package, not more local wrappers.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep `game.v1.CampaignAIService` as the authoritative cross-service contract, but introduce an AI-local collaborator seam before additional campaign-turn policy accumulates around raw generated clients.
  Rationale: The current session-grant and auth-state protocol is good. The coupling problem is that startup and service packages are binding directly to game proto clients and mixed availability semantics.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat stale validation commands as review findings, not just doc bugs.
  Rationale: If contributors cannot reliably invoke the relevant coverage, operational readiness is weaker even when the underlying tests exist.
  Date/Author: 2026-03-23 / Codex

- Decision: Sequence the implementation roadmap as deletion-first architecture batches with no long-lived compatibility shims.
  Rationale: The review goal explicitly allows breaking changes, and most of the structural debt now sits in seams that will only get worse if they are preserved during cleanup.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

Findings are grouped by review goal and classified as missing best practice, anti-pattern, or refactor candidate.

### Maintainability

1. Anti-pattern: shared internal gRPC metadata ownership currently belongs to the wrong service.
   Evidence:
   - `internal/services/ai/app/internal_identity.go`
   - `internal/services/ai/api/grpc/ai/campaign_context_helpers.go`
   - `internal/services/ai/orchestration/gametools/grpcctx.go`
   - all three import `internal/services/game/api/grpc/metadata`
   Why it matters:
   - AI cannot evolve its internal identity or campaign authority helpers without reaching into game transport internals.
   Refactor direction:
   - move request-id, campaign/session/participant headers, service-id helpers, and simple context accessors into one shared internal platform package.

2. Missing best practice: raw generated game clients are used directly across startup and service code without an AI-local collaborator seam.
   Evidence:
   - `internal/services/ai/app/server.go` dials `CampaignAIServiceClient` and `AuthorizationServiceClient` directly
   - `internal/services/ai/service/campaign_orchestration.go` and `internal/services/ai/service/usage_guard.go` encode AI policy directly against raw game RPCs
   Why it matters:
   - collaborator availability, staleness checks, and policy translation are spread across composition and service code instead of one explicit dependency boundary.
   Refactor direction:
   - introduce one AI-local game collaborator package or interface set that owns auth-state lookup, usage lookup, and campaign authorization checks.

3. Refactor candidate: optional game connectivity currently mixes silent degradation with later runtime failure.
   Evidence:
   - `dialGameService` in `internal/services/ai/app/server.go` returns nil clients when the optional managed connection is unavailable
   - `UsageGuard` then bypasses campaign usage checks when the collaborator is missing, while campaign orchestration later fails on missing `CampaignAIServiceClient`
   Why it matters:
   - the service has no single current-state policy for “game unavailable,” and contributors have to infer which workflows degrade vs fail closed.
   Refactor direction:
   - encode collaborator availability explicitly per workflow and make startup choose between “feature disabled” and “fail startup” instead of passing nil through the graph.

4. Missing best practice: the live debug-update contract is implicit and narrower than the docs suggest.
   Evidence:
   - `CampaignDebugUpdateBroker` is in-process best-effort fanout only
   - persisted debug traces are durable, but streaming updates are not a cross-process contract today
   Why it matters:
   - future worker/play integrations can easily over-assume real-time guarantees that do not exist.
   Refactor direction:
   - document the current guarantee honestly and defer any cross-process debug streaming until there is a dedicated contract.

### Testability

5. Missing best practice: AI integration verification guidance has been using stale test names and incomplete tag requirements.
   Evidence:
   - real AI integration tests live behind `//go:build integration`
   - replay tests live in `internal/test/integration/ai_campaign_context_replay_test.go`
   - live capture tests live behind `//go:build integration && liveai`
   - direct Daggerheart tool coverage currently uses `TestAIDirectSessionDaggerheartMechanicsTools` and `TestAIDirectSessionDaggerheartCombatFlowTools`
   Why it matters:
   - contributors can believe they validated AI integration behavior when they actually ran zero relevant tests.
   Refactor direction:
   - publish one canonical command set:
     - replay/direct coverage: `go test ./internal/test/integration -tags=integration -run 'TestAIDirectSessionDaggerheart(MechanicsTools|CombatFlowTools)|TestAIGMCampaignContextReplay'`
     - live capture: `go test ./internal/test/integration -tags=integration,liveai -run 'TestAIGMCampaignContextLiveCapture'`

6. Positive seam to preserve: debug-trace persistence and read-side access are AI-owned and already testable without game/worker storage reach-through.
   Evidence:
   - `internal/services/ai/debugtrace/`
   - `internal/services/ai/service/campaign_debug.go`
   - `internal/services/ai/storage/sqlite/store_campaign_debug_turns.go`
   Why it matters:
   - later operational work can build on this seam instead of introducing shared cross-service persistence shortcuts.
   Preservation note:
   - keep persisted trace ownership in AI even if UI/websocket forwarding expands elsewhere.

### Contributor Clarity

7. Missing best practice: the current cross-service story is better in architecture docs than in code navigation.
   Evidence:
   - `docs/architecture/platform/campaign-ai-orchestration.md` describes a clean game-issued grant contract
   - the code path still requires contributors to understand direct game metadata helpers, raw game RPC clients, and optional managed-connection behavior spread across multiple packages
   Why it matters:
   - new contributors read a boundary-oriented design, then immediately hit implementation ownership drift at the first cross-service call site.
   Refactor direction:
   - after the shared-metadata and collaborator-seam cleanup lands, update contributor docs so the code navigation matches the architectural story.

8. Positive seam to preserve: the campaign session-grant model and stale-grant validation rules are the right authority boundary.
   Evidence:
   - `docs/architecture/platform/campaign-ai-orchestration.md`
   - `internal/services/ai/service/campaign_orchestration.go`
   Why it matters:
   - this is one of the few cross-service contracts that already reads coherently in both docs and implementation.
   Preservation note:
   - keep the grant and auth-state contract stable while cleaning up helper placement and collaborator seams around it.

## Final Ranked Refactor Roadmap

The roadmap is ordered by dependency, not by local convenience.

1. Foundation fixes
   - Extract shared gRPC metadata and internal service-identity helpers out of `internal/services/game/api/grpc/metadata`.
   - Split AI startup/composition responsibilities so collaborator availability policy is explicit and feature-scoped.
   - Define one AI-local game collaborator seam for campaign auth state, binding usage, and campaign authorization.

2. Seam extraction
   - Shrink transport into stable auth/pagination/error helpers and stop routing more behavior through transport-local frameworks.
   - Split service-layer shared policies into smaller collaborators: auth-reference resolution, accessible-agent policy, usage policy, campaign-turn auth state.
   - Split provider capability interfaces and separate OpenAI invoke/model/OAuth responsibilities.

3. Storage and model cutovers
   - Add the storage/schema changes needed for a true `AuthReference` cutover.
   - Introduce a typed provider-connect session model and an atomic multi-write seam for coupled persistence workflows.
   - Remove storage-owned record DTO leakage where domain/support types should own the contract.

4. Runtime and orchestration cutovers
   - Replace the runner’s implicit completion policy with an explicit progression contract.
   - Move tool-result semantics onto a typed seam shared by orchestration and tool execution.
   - Split `gametools` into generic runtime tooling vs system-owned Daggerheart packages.

5. Quality and contributor cleanup
   - Replace `transport_test_helpers_test.go` with handler-family-local helpers and stronger service-level coverage.
   - Publish corrected AI integration commands and build tags in canonical docs.
   - Rewrite contributor and architecture docs so they stop blessing shrink-target seams.

6. Deletions after cutover
   - delete legacy `credential_id` / `provider_grant_id` exclusivity projections once `AuthReference` is persisted canonically
   - delete `storage.Store` if it remains unused after repository cleanup
   - delete dead provider fields like `TokenExchangeResult.LastRefreshError` and compatibility-only revoke paths
   - delete monolithic test/doc entrypoints that no longer represent stable seams

## Do First / Do Together / Do After Cutover

- Do first:
  - shared metadata/service-identity extraction
  - startup/composition cleanup
  - AI-local game collaborator seam
- Do together:
  - service-policy extraction with transport cleanup
  - storage/schema work with `AuthReference` cutover
  - orchestration progression contract with `gametools` decomposition
- Do after cutover:
  - contributor-doc rewrites
  - deletion of legacy projections, omnibus helpers, and stale test hubs

## Out of Scope but Required Collaborator Follow-Up

- Any change to `game.v1.CampaignAIService` or its grant payloads requires coordinated game-service work.
- Any future cross-process/live debug streaming guarantee requires explicit play/worker collaboration and should not be inferred from the current in-process broker.
- Worker retry/failure lifecycle policy was only reviewed at the AI-owned seam; deeper worker-domain redesign is out of scope for this review.

## Context and Orientation

Use:

- `docs/architecture/platform/campaign-ai-orchestration.md`
- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`

## Plan of Work

Inspect:

- game-service client usage and startup assumptions
- internal service identity and session-grant validation
- debug/update flow boundaries
- cross-service integration coverage depth
- ranked implementation order implied by prior findings

## Concrete Steps

1. Review AI-owned cross-service call sites and their tests.
2. Note any collaborator assumptions that are hidden rather than documented.
3. Gather the high-confidence findings from P01-P11 into one dependency-ordered list.
4. Classify roadmap items as foundation, seam extraction, cutover, deletion, or docs promotion.

## Validation and Acceptance

- `go test ./internal/services/ai/...`
- `go test ./internal/test/integration -tags=integration -run 'TestAIDirectSessionDaggerheart(MechanicsTools|CombatFlowTools)|TestAIGMCampaignContextReplay'`
- `go test ./internal/test/integration -tags=integration,liveai -run 'TestAIGMCampaignContextLiveCapture'`
- `make test`
- `make check`

Acceptance:

- AI-owned cross-service risks are explicit
- final roadmap has a clear execution order
- out-of-scope collaborator work is called out instead of implied

## Idempotence and Recovery

- If a prior pass is revised materially, rebuild the roadmap here rather than papering over stale assumptions.

## Artifacts and Notes

- The final ranked roadmap should also be copied into `00_master.md` once stabilized.

## Interfaces and Dependencies

Track any proposed changes to:

- session-grant and internal-identity contracts
- AI-owned game/worker client expectations
- final multi-pass roadmap sequencing
