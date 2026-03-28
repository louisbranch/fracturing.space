# P10: Test Architecture, Shared Fakes, Local Harnesses, and Integration Coverage

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the AI test strategy to determine whether tests live at the correct seam, whether shared fakes are actually shared, and whether contributors can add coverage without rebuilding bespoke harnesses.

Primary scope:

- `internal/test/mock/aifakes`
- AI package tests under `internal/services/ai/**`
- AI integration tests under `internal/test/integration`

## Progress

- [x] (2026-03-23 03:59Z) Inventoried AI test seams, shared fakes, local harnesses, and integration fixture entrypoints.
- [x] (2026-03-23 04:48Z) Recorded findings and proposed target test-support architecture.
- [x] (2026-03-23 04:49Z) Validation complete: `go test ./internal/services/ai/...` passed.
- [x] (2026-03-23 04:50Z) Checked current integration guidance: `go test ./internal/test/integration -run 'TestAIDirectSessionDaggerheart|TestAIGMCampaignContextReplay|TestAIGMCampaignContextLiveCapture'` returned `[no tests to run]` because AI integration tests are build-tagged and live-capture coverage also requires `liveai`.

## Surprises & Discoveries

- The codebase already contains both the good pattern and the bad pattern. Orchestration tests mostly use tiny seam-local fakes, while transport tests rely on a very large custom helper harness.
- `internal/test/mock/aifakes` is capability-scoped, but several fakes contain enough behavioral logic to act like a second in-memory implementation of storage semantics.
- Service-layer package coverage is surprisingly thin relative to the amount of service behavior exercised through handler tests.
- The documented AI integration commands are stale because of build tags, not only because of test-name drift.
- There are no fuzz/property-style tests anywhere in the AI subtree right now.

## Decision Log

- Decision: Preserve seam-local fakes in packages like `orchestration` and use them as the model for future refactors.
  Rationale: These tests are easier to read, cheaper to extend, and clearly tied to one package contract. They embody the testing policy better than the transport-local harness framework.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat `transport_test_helpers_test.go` as a shrink target rather than a stable test framework.
  Rationale: The file now bundles fake stores, fake adapters, handler constructors, and type-assertion wiring across multiple RPC families. That centralization makes transport tests harder to reason about and slows contributor edits.
  Date/Author: 2026-03-23 / Codex

- Decision: Keep `aifakes` for genuinely shared repository seams, but stop promoting behavior-heavy test logic into that package by default.
  Rationale: Once a fake starts reproducing pagination, uniqueness, or workflow-specific mutation behavior, it becomes another implementation that can drift from the concrete adapter.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

Findings are grouped by review goal and classified as missing best practice, anti-pattern, or refactor candidate.

### Maintainability

1. Anti-pattern: transport tests depend on a large bespoke harness instead of small seam-local helpers.
   Evidence:
   - `internal/services/ai/api/grpc/ai/transport_test_helpers_test.go` is 718 lines.
   - It defines a composite `fakeStore`, multiple fake adapters/clients, and many `new...HandlersWithOpts` constructors.
   Why it matters:
   - Contributors changing one handler family inherit a broad, shared test framework with cross-family assumptions.
   Refactor direction:
   - Split helper builders by handler family or move more behavior testing down into service/package tests so transport helpers only cover transport concerns.

2. Missing best practice: service-layer coverage is too thin relative to business-logic surface area.
   Evidence:
   - `internal/services/ai/service/` currently has only 3 test files.
   - transport has 12 test files and many handlers construct real services through the transport helper layer.
   Why it matters:
   - Durable workflow behavior for agent, credential, access-request, invocation, and provider-grant flows is still proved heavily through transport tests instead of the service seam that owns it.
   Refactor direction:
   - Add direct service-package tests for workflow logic and trim transport tests back toward authz/request-mapping/error-mapping coverage.

3. Refactor candidate: shared `aifakes` already contain non-trivial storage semantics.
   Evidence:
   - `aifakes/agents.go` and `aifakes/credentials.go` enforce normalized-label conflict behavior.
   - `aifakes/access_requests.go` reproduces review/revoke state transitions.
   - `aifakes/audit_events.go` implements filtered pagination behavior.
   Why it matters:
   - These are no longer “dumb” fakes; they are alternate implementations that can drift from sqlite contracts and consume maintenance time.
   Refactor direction:
   - Keep only truly shared capability seams in `aifakes`; move behavior-heavy logic back to package-local fakes when only one test area needs it.

4. Refactor candidate: transport tests still rely on composite fake stores and interface discovery by type assertion.
   Evidence:
   - `newAgentHandlersWithOpts`, `newInvocationHandlersWithOpts`, and related helpers discover extra repositories by type-asserting the same fake values to other interfaces.
   - `fakeStore` embeds many `aifakes` repositories and overrides `ListAccessibleAgents`.
   Why it matters:
   - Test setup mirrors the codebase’s broad dependency bags instead of forcing explicit seam ownership.
   Refactor direction:
   - Build smaller handler test fixtures with explicit collaborators per RPC family and delete the omnibus fake-store pattern.

### Testability

5. Missing best practice: integration guidance is stale because build tags are not encoded in the advertised command.
   Evidence:
   - `go test ./internal/test/integration -run 'TestAIDirectSessionDaggerheart|TestAIGMCampaignContextReplay|TestAIGMCampaignContextLiveCapture'` returned `[no tests to run]`.
   - AI integration files use `//go:build integration`, and live-capture tests use `//go:build integration && liveai`.
   Why it matters:
   - Contributors following the documented command will think there is AI integration coverage when none actually ran.
   Refactor direction:
   - Update validation guidance to include `-tags=integration`, and document `liveai` separately for real-model capture tests.

6. Missing best practice: there are no fuzz/property-style tests across AI contracts.
   Evidence:
   - repository search found no `Fuzz`, `quick.Check`, or property-test usage in `internal/services/ai` or AI integration tests.
   Why it matters:
   - Several AI seams are parser/formatter heavy: page tokens, artifact paths, instruction loading, memory section edits, and tool/result JSON normalization.
   Refactor direction:
   - Add fuzz/property tests for artifact path normalization, memory section round-trips, debug-turn page-token parsing, and selected proto/helper normalization functions.

7. Refactor candidate: integration coverage mixes replay, live-capture, and broader game end-to-end suites without a crisp AI-owned entrypoint.
   Evidence:
   - `internal/test/integration/` contains dedicated AI replay/live files, AI tool files, and the broader `TestGameEndToEnd` shared suite.
   - helper infrastructure is split across `harness_test.go`, `ai_campaign_context_helpers_test.go`, and `ai_openai_fixture_test.go`.
   Why it matters:
   - Contributors need to understand a large shared integration package before they can tell which tests specifically guard AI contracts.
   Refactor direction:
   - Keep shared fixture wiring, but document or group AI-owned integration entrypoints more explicitly around replay, direct-session tools, and optional live-capture flows.

8. Positive seam to preserve: orchestration tests use small package-local doubles effectively.
   Evidence:
   - `orchestration/runner_test.go`, `prompt_builder_test.go`, and `daggerheart/context_sources_test.go` define tight fake sessions, providers, and collectors locally.
   Why it matters:
   - These tests are readable and align closely with package-owned contracts.
   Preservation note:
   - Prefer this style over expanding shared fake packages for orchestration concerns.

### Contributor Clarity

9. Missing best practice: the current test layout makes it hard to know where a new behavior test belongs.
   Evidence:
   - contributor docs recommend seam-local tests, but transport has the biggest AI test hub and a reusable helper framework that invites more additions.
   - service has only 3 direct test files despite many workflow services.
   Why it matters:
   - New contributors are likely to add tests where helpers already exist rather than where the behavior is actually owned.
   Refactor direction:
   - Update contributor docs after cleanup with a package-to-test-level matrix that pushes workflow logic down to service/domain packages and reserves transport helpers for public contract checks.

10. Positive seam to preserve: `aifakes` is capability-scoped rather than one single omnibus fake package.
    Evidence:
    - separate fake files for credentials, agents, access requests, provider grants, artifacts, connect sessions, audit events, and sealer.
    Why it matters:
    - The package already has the right granularity directionally.
    Preservation note:
    - Preserve capability-specific organization even if some fakes are moved back to local test files.

Package-to-test-level matrix after cleanup:

- domain packages: package-local invariant tests
- service packages: workflow, auth resolution, usage guard, and error-path tests with small local or shared repository fakes
- transport package: request parsing, auth extraction, pagination validation, and status mapping tests only
- orchestration/gametools: package-local runtime and tool contract tests with local fakes
- sqlite: package-local persistence and migration contract tests
- integration: AI replay/direct-session contract tests behind `-tags=integration`; live-capture tests behind `-tags='integration liveai'`

Concrete refactor slices for a later implementation batch:

1. Split `transport_test_helpers_test.go` by handler family or replace it with smaller helper files that do not share one composite fake-store abstraction.
2. Add direct service tests for workflow-heavy services currently covered mostly through handlers.
3. Move behavior-heavy fake logic out of `aifakes` when it is only needed in one test area.
4. Publish a canonical AI integration command set with explicit build tags and separate live-capture guidance.
5. Add a first wave of fuzz/property tests for parser/normalizer seams.

Tests to add, move, or delete in the refactor phase:

- Add service-level tests for `AgentService`, `CredentialService`, `AccessRequestService`, `InvocationService`, and `ProviderGrantService` workflow behavior.
- Keep transport tests, but delete or slim cases whose only purpose is to prove business logic already covered below transport.
- Add fuzz/property tests for artifact paths, memorydoc section transforms, and debug-trace page tokens.
- Update integration commands and add narrow AI-owned wrappers if contributors need easier `-run` entrypoints.

Docs to update in the refactor phase:

- `docs/reference/ai-service-contributor-map.md`
- `docs/architecture/policy/testing-policy.md` only if AI-specific guidance needs durable mention
- any AI-specific running/reference docs that currently show stale integration commands

## Context and Orientation

Use:

- `docs/architecture/policy/testing-policy.md`
- `docs/reference/ai-service-contributor-map.md`
- `internal/test/mock/aifakes/doc.go`

## Plan of Work

Inspect:

- reuse vs duplication between `aifakes` and local test helpers
- whether business logic is over-tested through transport
- package-vs-integration test balance
- missing property/fuzz opportunities
- scenario/integration coverage around orchestration and campaign context
- contributor cost of setting up realistic tests

## Concrete Steps

1. Inventory shared fakes and local fake/harness types.
2. Note which local helpers deserve promotion, shrinking, or deletion.
3. Map key behaviors to the seam where they are currently tested.
4. Produce a package-to-test-level matrix and fake ownership plan.

## Validation and Acceptance

- `go test ./internal/services/ai/...`
- `go test ./internal/test/integration -run 'TestAIDirectSessionDaggerheart|TestAIGMCampaignContextReplay|TestAIGMCampaignContextLiveCapture'`

Acceptance:

- target fake/harness ownership is explicit
- package-level vs integration-level coverage guidance is explicit
- test deletion or migration candidates are named concretely

## Idempotence and Recovery

- Prefer seam-local fakes over heavyweight shared fixtures unless many packages genuinely share the same repository boundary.

## Artifacts and Notes

- Coverage gaps that should block large refactors if left unresolved:
  - thin direct service-layer workflow coverage
  - stale AI integration commands that do not actually execute tagged tests
  - no fuzz/property coverage for parser/normalizer-heavy AI seams

## Interfaces and Dependencies

Track any proposed changes to:

- `internal/test/mock/aifakes` APIs
- transport test helper surfaces
- integration fixture contracts
- build-tagged integration command guidance
