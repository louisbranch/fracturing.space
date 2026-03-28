# AI Service Comprehensive Review Program

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Drive a comprehensive AI-service review that surfaces maintainability, testability, and contributor-clarity issues across the full AI boundary, not just a handful of obvious findings. The output is a ranked refactor program that is comfortable making breaking changes when they simplify architecture and contributor experience.

This review covers:

- `cmd/ai` and `internal/cmd/ai`
- `api/proto/ai/v1/service.proto`
- `internal/services/ai/**`
- AI contributor/reference docs
- AI-specific shared fakes and integration tests
- Direct AI-owned seams into game and worker

## Progress

- [x] (2026-03-23 03:59Z) Established scope and pass breakdown; created master ExecPlan and child pass plans under `.agents/plans/review-ai/`.
- [x] (2026-03-23 03:59Z) Baseline discovery complete: reviewed architecture foundations, AI architecture/orchestration/agent-system/mechanics docs, contributor map, lifecycle terms, package inventory, RPC inventory, doc.go coverage, shared AI fakes, and AI integration-test inventory.
- [x] (2026-03-23 03:59Z) Baseline validation complete: `go test ./internal/services/ai/...` passed.
- [x] (2026-03-23 04:00Z) Started P01 and captured initial startup-boundary findings around duplicated runtime validation, monolithic handler assembly, manual service-registration drift risk, and cross-service metadata helper placement.
- [x] (2026-03-23 04:08Z) Completed P01 target runtime shape and cutover order.
- [x] (2026-03-23 04:08Z) Started P02 and captured initial transport findings around repeated unary auth scaffolding, low-signal root-file sprawl, multi-dialect error mapping, campaign-context validator coupling, and transport test harness weight.
- [x] (2026-03-23 04:13Z) Completed P02 target transport helper surface and cutover order.
- [x] (2026-03-23 04:18Z) Started P03 and captured initial service-layer findings around overly broad shared helpers, duplicated auth-reference policy in `AgentService`, production test-setter methods on `AccessRequestService`, and mixed helper error conventions.
- [x] (2026-03-23 04:21Z) Completed P03 target service-layer helper split and cutover order.
- [x] (2026-03-23 04:27Z) Started P04 and captured domain-lifecycle findings around package-comment drift, typed-auth-reference edge projections, credential secret-boundary ambiguity, access-request revoke vocabulary, and debug-trace test gaps.
- [x] (2026-03-23 04:33Z) Completed P04 target domain-boundary cleanup and cutover order.
- [x] (2026-03-23 04:39Z) Started P05 and captured provider-boundary findings around credential-shaped generic auth vocabulary, OpenAI-shaped model metadata, omnibus adapter responsibilities, optional revoke semantics, and Responses/schema test gaps.
- [x] (2026-03-23 04:48Z) Completed P05 target provider seam cleanup and cutover order.
- [x] (2026-03-23 04:53Z) Started P06 and captured orchestration findings around the runner’s implicit state machine, tool-name/result-hint coupling, silent degraded defaults, typed-state ownership in the context registry, and coarse observability.
- [x] (2026-03-23 05:03Z) Completed P06 target orchestration pipeline cleanup and cutover order.
- [x] (2026-03-23 05:08Z) Started P07 and captured game-tool findings around monolithic registry ownership, generic-vs-Daggerheart boundary drift, hidden turn-progression output contracts, and Daggerheart context-source JSON coupling.
- [x] (2026-03-23 05:13Z) Completed P07 target generic-vs-system tool boundary and refactor order.
- [x] (2026-03-23 05:19Z) Started P08 and captured campaign-context findings around split instruction fallback ownership, Daggerheart-shaped system registration, storage-shaped artifact vocabulary, and reference-corpus repo-layout coupling.
- [x] (2026-03-23 05:21Z) Completed P08 target ownership map for artifacts, instructions, memory, and references.
- [x] (2026-03-23 05:29Z) Started P09 and captured storage findings around mixed domain-vs-record contracts, legacy auth-reference persistence, raw-string provider-connect sessions, duplicated sqlite pagination/validation logic, and missing transaction seams.
- [x] (2026-03-23 05:36Z) Completed P09 target storage seam cleanup and validation.
- [x] (2026-03-23 05:43Z) Started P10 and captured test-architecture findings around the oversized transport helper harness, behavior-heavy shared fakes, thin direct service coverage, and stale build-tagged integration commands.
- [x] (2026-03-23 05:50Z) Completed P10 target fake/harness architecture and validation notes.
- [x] (2026-03-23 05:56Z) Started P11 and captured contributor-clarity findings around architecture-doc overstatements, stale extension guidance, package/directory naming friction at the composition root, and uneven trustworthiness across AI reference docs.
- [x] (2026-03-23 06:01Z) Completed P11 durable doc cleanup set and contributor-routing updates.
- [x] (2026-03-23 06:08Z) Started P12 and captured cross-service findings around game-owned metadata helper leakage, raw game-client usage, mixed game-unavailable behavior, in-process-only debug updates, and stale integration command guidance.
- [x] (2026-03-23 06:16Z) Completed P12 and consolidated the final ranked refactor roadmap.
- [x] P01 complete: runtime composition and startup boundary
- [x] P02 complete: gRPC transport, auth extraction, proto mapping, and error normalization
- [x] P03 complete: service workflow boundaries, dependency seams, authorization policy, and error taxonomy
- [x] P04 complete: domain lifecycle packages and typed invariants
- [x] P05 complete: provider abstraction and OpenAI adapter isolation
- [x] P06 complete: orchestration core
- [x] P07 complete: game tools and Daggerheart-specific AI seams
- [x] P08 complete: campaign context, instruction loading, memory/reference corpus, and artifact policy
- [x] P09 complete: storage contracts, sqlite adapter layout, migrations, and pagination/filter contracts
- [x] P10 complete: test architecture
- [x] P11 complete: contributor clarity
- [x] P12 complete: cross-service seams and consolidated roadmap

## Surprises & Discoveries

- The AI subtree is broad enough that the review must be pass-based rather than package-by-package only: 22 Go packages, 8 public gRPC services, and 25 RPCs.
- The current AI baseline is stable enough to support review-driven refactors: `go test ./internal/services/ai/...` passed before any code changes.
- The test-support story is mixed: `internal/test/mock/aifakes` exists and is used, but transport tests still carry a large local harness surface.
- The first startup pass already surfaced one cross-service boundary leak: AI startup imports request-metadata helpers from `internal/services/game/api/grpc/metadata` instead of a shared package.
- The same cross-service metadata leak appears in AI transport via `campaign_context_helpers.go`, which suggests one shared fix can remove two separate boundary violations.
- The service layer’s main architectural pressure is not too many workflow services; it is helpers whose names imply narrow concerns while their implementations perform broader orchestration and mutation.
- The domain layer is materially healthier than the outer layers: most AI lifecycle transitions already live in the right packages, so later refactors should delete edge duplication instead of reorganizing domain ownership.
- `agent.AuthReference` is the main cross-pass cleanup thread now: the domain owns it cleanly, but transport and sqlite still project it through nullable credential/grant pairs.
- `providergrant` is the model lifecycle package to preserve: refresh-state transitions are already explicit and package-local.
- The provider boundary is “generic” mostly in naming, not in shape: the current shared contracts still assume OpenAI-style auth material and model metadata.
- One OpenAI concrete type currently spans service invocation, model listing, and orchestration runtime, which is too much capability for one adapter seam.
- OpenAI strict-schema policy is correctly provider-local, but its test coverage is indirect and weaker than the risk profile of that compatibility helper.
- The orchestration package has the right major building blocks, but the runner is carrying too much turn-completion policy inline.
- The typed `SessionBrief` path is one of the healthiest seams in the AI service and should be preserved during later refactors.
- The next orchestration cleanup thread is explicit progression policy: tool-name constants and JSON result-hint parsing currently live in the core runner.
- `gametools` is broader than its name and should not be treated as the long-term extension seam for new systems: generic session plumbing, artifact/reference tools, Daggerheart read surfaces, and Daggerheart mechanics are still mixed together.
- The current tool/result contract is split across packages without an explicit type: interaction tools emit JSON hints and the runner consumes them as runtime policy.
- `internal/services/ai/orchestration/daggerheart` is already the strongest system-specific seam in this area and should become the model for later extractions, not be folded back into generic packages.
- The P07 validation target exposed a test-gap signal: `go test ./internal/test/integration -run 'TestAIDirectSessionDaggerheart'` currently matches no tests.
- The package split inside `campaigncontext` is mostly healthy, but the composition edge is still Daggerheart-first: startup, instruction resolution, and reference-corpus support are not yet encoded as a real multi-system extension seam.
- Instruction fallback policy is currently split between `campaigncontext` and `app/server.go`, even though the docs describe instruction files as the canonical behavior surface.
- `memorydoc` is one of the healthier AI support packages and should remain narrow.
- The campaign-context validation command in the master plan is also stale right now: `go test ./internal/test/integration -run 'TestAIGMCampaignContextReplay|TestAIGMCampaignContextLiveCapture'` matches no tests as written.
- The storage package is internally inconsistent today: some repository seams are cleanly domain-owned, while others still expose storage-owned record DTOs and raw string lifecycle fields.
- The architecture doc overstates storage/domain alignment: the repo still has separate storage record types for artifacts, audit events, and provider connect sessions.
- The sqlite adapter is split by aggregate file, which is good, but shared pagination/validation mechanics are still largely copy-pasted.
- There is no transaction seam for coupled persistence workflows, which means some service flows already rely on multi-call best effort rather than explicit atomicity.
- The auth-reference cleanup thread remains blocked by the current `ai_agents` schema shape in sqlite.
- The best AI tests already demonstrate the desired pattern: orchestration tests use small package-local doubles and are easier to evolve than the transport-layer helper framework.
- Transport is still the largest AI test hub, with a 718-line shared helper file that effectively acts as a local test framework.
- Shared `aifakes` are useful, but several now reproduce meaningful storage/workflow behavior and are at risk of drifting from real contracts.
- The documented AI integration commands are stale because build tags are missing, and live-capture coverage also needs a separate `liveai` tag.
- There are currently no fuzz/property-style tests anywhere in the AI subtree.
- The AI docs are broadly present and better than average, but several of them still overstate architectural cleanliness or point contributors straight at shrink-target implementation files.
- The contributor map’s overall reading order is good; the drift is in the specifics, especially test guidance and tool-extension guidance.
- The package/directory mismatch at `internal/services/ai/app` (`package server`) is minor but repeatedly confusing in contributor-facing guidance.
- Secondary reference docs like `ai-tools.md` and `ai-resources.md` currently amplify outdated monolithic `gametools` assumptions rather than clarifying stable contracts.
- The session-grant contract itself is not the weak point; the cross-service debt is mostly helper ownership, raw generated-client usage, and mixed collaborator-availability policy around it.
- AI still imports game-owned gRPC metadata helpers from three separate seams: startup/internal identity, transport campaign-context validation, and gametool outgoing context assembly.
- The AI-owned debug-trace storage path is structurally cleaner than the live-update path; persisted traces are durable, but real-time updates are only in-process best-effort today.
- The real AI integration suite is healthier than the earlier plan commands suggested, but its invocation surface is tag-sensitive and the previously documented command names were stale.

## Decision Log

- Decision: Use one master ExecPlan plus one child ExecPlan per review pass.
  Rationale: The review needs a single source of truth for tally/status, but each area also needs a self-contained document with enough context for follow-up implementation work.
  Date/Author: 2026-03-23 / Codex

- Decision: Include immediate AI-owned seams, docs, shared fakes, and integration coverage in scope.
  Rationale: The stated goal is contributor readiness, which depends on contracts, docs, and test harnesses as much as production packages.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep the current AI domain package split and focus later refactors on edge-contract cleanup rather than domain-package reorganization.
  Rationale: P04 found that `agent`, `credential`, `providergrant`, and `accessrequest` mostly own their lifecycle rules correctly already. The structural debt is primarily at the storage/transport edges and in vocabulary drift.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep `internal/services/ai/provider` as a small shared vocabulary package, but treat capability interfaces and model/auth payloads as shrink targets rather than extension points.
  Rationale: P05 found that the current shared provider seam is broader than the real product contract and carries OpenAI-shaped details that should not harden into long-term generic API surface.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep the orchestration collector/renderer split and typed session brief, and focus refactors on explicit turn-progression policy around the runner.
  Rationale: P06 found that prompt collection/rendering is already a good seam. The structural debt is in runtime completion policy and context/tool control semantics encoded inline in `runner.go`.
  Date/Author: 2026-03-23 / Codex

- Decision: Preserve `internal/services/ai/orchestration/daggerheart` as the Daggerheart-owned prompt/context seam and treat `internal/services/ai/orchestration/gametools` as a shrink target.
  Rationale: P07 found that Daggerheart prompt assembly already has a clearer system-specific home than the mixed tool stack. Future refactors should move more system-specific authority toward dedicated system packages rather than continuing to extend generic `gametools`.
  Date/Author: 2026-03-23 / Codex

- Decision: Refactor tool-result progression semantics together with the orchestration runner rather than as an isolated game-tool cleanup.
  Rationale: P06 and P07 both found the same hidden contract from different sides. The clean fix is one explicit shared seam, not duplicated local cleanups.
  Date/Author: 2026-03-23 / Codex

- Decision: Consolidate instruction composition and missing-file degradation into one `instructionset`-owned seam.
  Rationale: P08 found the current behavior split between inline fallback markdown in `campaigncontext` and partial-load policy in `app/server.go`. A contributor should only need one place to understand how prompt instructions are resolved.
  Date/Author: 2026-03-23 / Codex

- Decision: Do not harden the current `referencecorpus` shape as the long-term generic system-reference abstraction without a deliberate follow-up design choice.
  Rationale: P08 found a mismatch between generic naming and Daggerheart-specific behavior, including single-system validation and repo-local playbook loading. Future work should either specialize the seam honestly or make it genuinely multi-system.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep direct sqlite-to-domain scanning for core AI aggregates, but treat storage-owned record DTOs as temporary seams to shrink or promote.
  Rationale: P09 found the cleanest persistence paths are the ones that reconstruct canonical domain types directly. The confusing seams are the mixed support-workflow contracts that still expose storage-local records and raw string fields.
  Date/Author: 2026-03-23 / Codex

- Decision: Add an explicit transaction or atomic-write seam before doing larger persistence cleanups that span multiple records.
  Rationale: P09 found at least one live workflow (`FinishConnect`) that already performs coupled writes without an atomic repository boundary. Later refactors should not cement that pattern further.
  Date/Author: 2026-03-23 / Codex

- Decision: Move more workflow behavior coverage down to service packages and shrink transport tests back toward transport concerns.
  Rationale: P10 found the codebase’s actual easiest path for adding tests is still the transport helper framework, even though the documented testing policy prefers seam-local coverage. The fix is to make the intended seams cheaper and clearer than the accidental one.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep shared `aifakes` capability-scoped, but resist turning them into a second authoritative implementation of storage or workflow behavior.
  Rationale: P10 found that some fakes already encode normalized-label conflicts, filtered pagination, and review/revoke transitions. Shared fakes should support tests, not quietly become another semantics layer to maintain.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep the contributor map as the main reader-first AI routing doc, but remove unstable implementation-file recommendations from it.
  Rationale: P11 found that the reading order and broad routing are good. The problem is that the document currently normalizes shrink-target files like `transport_test_helpers_test.go` and monolithic `gametools` registration surfaces.
  Date/Author: 2026-03-23 / Codex

- Decision: Prefer honest current-state documentation over aspirational architecture wording where cutovers are still incomplete.
  Rationale: P11 found that several contributor-facing mismatches come from docs that describe the desired end state as if it already fully exists, especially around storage contracts and extension seams.
  Date/Author: 2026-03-23 / Codex

- Decision: Extract shared internal gRPC metadata and service-identity helpers into a platform/shared package before doing deeper AI/game seam cleanup.
  Rationale: P01, P02, and P12 found the same ownership leak in runtime, transport, and orchestration code. One shared substrate fix removes multiple boundary violations at once.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep `game.v1.CampaignAIService` as the source-of-truth collaborator contract, but add an AI-local collaborator seam around raw generated clients before more policy accumulates.
  Rationale: P12 found the grant/auth-state protocol is coherent, but availability and policy translation are currently spread across composition and service packages.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat stale validation commands as operational-readiness issues, not minor doc drift.
  Rationale: P07, P08, P10, and P12 all found examples where the documented command surface did not actually exercise the intended AI coverage.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

The review is complete. The findings are not evenly distributed: the domain packages are in relatively good shape, while the highest-value cleanup sits in edge seams, collaborator boundaries, storage cutovers, orchestration runtime policy, and contributor/test guidance.

## Ranked Refactor Roadmap

1. Foundation fixes
   - extract shared gRPC metadata and internal service-identity helpers out of the game transport tree
   - clean up AI startup/composition so collaborator availability is explicit and feature-scoped
   - introduce one AI-local game collaborator seam for campaign auth state, binding usage, and authorization checks

2. Seam extraction
   - shrink transport into stable auth/pagination/error helpers and stop growing transport-local test infrastructure
   - split service-layer shared policies into smaller collaborators with clearer ownership
   - split provider capability interfaces and separate OpenAI invoke/model/OAuth responsibilities

3. Storage and model cutovers
   - land the schema and repository changes needed for a true `AuthReference` cutover
   - add a typed provider-connect session model and an atomic multi-write seam
   - remove storage-owned record DTO leakage where domain/support vocabularies should own the contract

4. Runtime and orchestration cutovers
   - replace the runner’s implicit completion policy with an explicit progression contract
   - move tool-result semantics onto a typed seam shared by orchestration and tool execution
   - split `gametools` into generic runtime tooling vs system-owned Daggerheart behavior

5. Quality and contributor cleanup
   - replace `transport_test_helpers_test.go` with handler-family-local helpers and stronger service coverage
   - publish corrected AI integration commands and build tags in canonical docs
   - rewrite contributor and architecture docs so they stop blessing shrink-target seams

6. Deletions after cutover
   - delete legacy `credential_id` / `provider_grant_id` auth-reference projections once canonical persistence lands
   - delete unused omnibus seams such as `storage.Store` if they remain redundant
   - delete dead provider fields and compatibility-only paths
   - delete monolithic doc/test entrypoints that no longer represent stable seams

## Foundational vs Optional

Foundational:
- shared metadata/service-identity extraction
- startup/composition cleanup
- AI-local game collaborator seam
- `AuthReference` storage cutover
- orchestration progression contract
- `gametools` decomposition

Important but secondary:
- provider contract cleanup
- campaign-context/reference ownership cleanup
- transport harness reduction
- contributor-doc rewrites after code cutovers

Optional polish after structural work:
- minor package-comment recalibration
- naming cleanup where the package boundary is already otherwise healthy

## Intentionally Out of Scope

- deep redesign of worker-domain retry/failure lifecycle beyond the AI-owned seam
- any unilateral change to `game.v1.CampaignAIService` or grant payloads without coordinated game-service work
- cross-process/live debug streaming guarantees beyond the current AI-owned persisted trace plus in-process update model

## Context and Orientation

Use these documents as the canonical architecture baseline before changing any plan:

- `docs/architecture/foundations/architecture.md`
- `docs/architecture/foundations/domain-language.md`
- `docs/architecture/policy/testing-policy.md`
- `docs/architecture/platform/ai-service-architecture.md`
- `docs/architecture/platform/campaign-ai-orchestration.md`
- `docs/architecture/platform/campaign-ai-agent-system.md`
- `docs/architecture/platform/campaign-ai-mechanics-quality.md`
- `docs/reference/ai-service-contributor-map.md`
- `docs/reference/ai-service-lifecycle-terms.md`

Child plans own the detailed package-level reading lists.

## Plan of Work

Execute the review in this order:

1. Runtime and boundary passes first: P01-P03.
2. Core model/runtime passes next: P04-P09.
3. Quality and contributor passes after that: P10-P11.
4. Cross-service synthesis and final roadmap last: P12.

Each child plan must produce:

- findings grouped by maintainability, testability, and contributor clarity
- explicit best practices missing or anti-patterns present
- a concrete refactor direction with deletion candidates
- tests to add, move, or delete
- docs to update or remove
- public/exported interface impact

## Concrete Steps

1. Keep the progress tally current here after every child-pass update.
2. Treat each child plan as the working notebook for one bounded review pass.
3. Promote durable cross-pass decisions into this file's `Decision Log`.
4. When a pass uncovers a likely breaking refactor, record the intended cutover order here so later passes can align to it.
5. End by collapsing all pass findings into one sequenced roadmap: foundation fixes, seam extractions, cutovers, deletions, docs promotion.

## Validation and Acceptance

- Baseline command: `go test ./internal/services/ai/...`
- Package-local refactor validation: targeted `go test` commands named in the child plan
- AI runtime contract validation when relevant:
  `go test ./internal/test/integration -tags=integration -run 'TestAIDirectSessionDaggerheart(MechanicsTools|CombatFlowTools)|TestAIGMCampaignContextReplay'`
- Live-capture validation when relevant:
  `go test ./internal/test/integration -tags=integration,liveai -run 'TestAIGMCampaignContextLiveCapture'`
- Final verification before shipping a refactor batch: `make test` then `make check`

The review program is complete when:

- every in-scope AI-owned area has one and only one child plan
- every child plan has a finished `Progress` trail and acceptance notes
- every proposed breaking change has rationale plus cutover order
- this master file contains a consolidated, ranked refactor roadmap

## Idempotence and Recovery

- Re-reading repo docs and package entrypoints is safe and expected before updating a pass.
- If a child pass changes shape materially, replace its plan content rather than layering contradictory notes.
- If findings from one pass invalidate another pass's assumptions, update both child plans and record the dependency in this master file.

## Artifacts and Notes

- Child plans live in `.agents/plans/review-ai/01_*.md` through `12_*.md`.
- Existing `.agents/plans/review/` artifacts are for the game service and should not be treated as AI findings.

## Interfaces and Dependencies

Track interface impact explicitly in every child plan for:

- `api/proto/ai/v1/service.proto`
- exported Go contracts in `service`, `storage`, `provider`, `orchestration`, and domain packages
- AI contributor/reference docs when terminology or ownership changes
- shared test-support APIs in `internal/test/mock/aifakes`
