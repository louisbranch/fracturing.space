---
title: "AI service contributor map"
parent: "Reference"
nav_order: 17
status: canonical
owner: engineering
last_reviewed: "2026-03-23"
---

# AI service contributor map

Reader-first routing guide for contributors changing the AI service.

## Start here

Read in this order:

1. [Architecture foundations](../architecture/foundations/index.md)
2. [Campaign AI orchestration](../architecture/platform/campaign-ai-orchestration.md)
3. [Campaign AI session bootstrap](../architecture/platform/campaign-ai-session-bootstrap.md)
4. [Campaign AI agent system](../architecture/platform/campaign-ai-agent-system.md)
5. [Campaign AI mechanics quality](../architecture/platform/campaign-ai-mechanics-quality.md)
6. [AI service lifecycle terms](ai-service-lifecycle-terms.md)
7. This page

Use [Verification commands](../running/verification.md) for the canonical local
check sequence.

## Where to edit

| Change you want | Primary packages/files |
| --- | --- |
| Change process startup, runtime config, or service registration | `internal/cmd/ai/`, `internal/services/ai/app/` |
| Change gRPC handler seams, auth checks, or proto mapping | `internal/services/ai/api/grpc/ai/` |
| Change use-case orchestration, auth token resolution, access control, or usage policy | `internal/services/ai/service/` |
| Change agent, credential, provider-grant, or access-request domain rules | `internal/services/ai/agent/`, `internal/services/ai/credential/`, `internal/services/ai/providergrant/`, `internal/services/ai/accessrequest/` |
| Change provider identity, provider bundle registration, or provider-reported usage contracts | `internal/services/ai/provider/`, `internal/services/ai/providercatalog/`, `internal/services/ai/app/runtime_deps.go` |
| Change provider OAuth handshake contracts, optional revoke capability, or connect-session lifecycle typing | `internal/services/ai/provideroauth/`, `internal/services/ai/providerconnect/`, `internal/services/ai/service/provider_grant.go`, `internal/services/ai/service/provider_grant_runtime.go`, `internal/services/ai/storage/sqlite/` |
| Change OpenAI or Anthropic invocation/model listing behavior, or provider-specific HTTP translation | `internal/services/ai/provider/openai/`, `internal/services/ai/provider/anthropic/` |
| Change campaign-turn orchestration, prompt assembly, tool dispatch, or provider step aggregation | `internal/services/ai/orchestration/`, `internal/services/ai/orchestration/gametools/` for the generic direct-session shell and registry, `internal/services/ai/orchestration/daggerhearttools/` for Daggerheart-specific tool/resource execution, and `internal/services/ai/orchestration/daggerheart/` for current system-specific prompt context |
| Change campaign artifact bootstrapping or artifact path policy | `internal/services/ai/campaigncontext/`, `internal/services/ai/api/grpc/ai/*artifact*` |
| Change AI instruction-file loading, memory document structure, or system reference corpus logic | `internal/services/ai/campaigncontext/instructionset/`, `internal/services/ai/campaigncontext/memorydoc/`, `internal/services/ai/campaigncontext/referencecorpus/`, `internal/services/ai/api/grpc/ai/*reference*` |
| Change auth-reference typing, provider-grant refresh transitions, or lifecycle vocabulary | `internal/services/ai/agent/`, `internal/services/ai/providergrant/`, [ai-service-lifecycle-terms.md](ai-service-lifecycle-terms.md) |
| Change SQLite persistence or migration-backed aggregate storage contracts | `internal/services/ai/storage/sqlite/`, `internal/services/ai/storage/` |
| Add AI test doubles or handler test setup | `internal/test/mock/aifakes/`, plus handler-family-local `*_test_support_test.go` files under `internal/services/ai/api/grpc/ai/` |
| Change cross-service campaign-turn or replay/integration coverage | `internal/test/integration/`, `internal/services/worker/domain/` |

## Key lifecycle terms

Use [AI service lifecycle terms](ai-service-lifecycle-terms.md) as the
canonical vocabulary for:

- auth references on agents
- provider-grant refresh success and failure states
- typed session briefs and bootstrap mode
- prompt render policy at the composition root

## Package reading order

1. `internal/services/ai/app/`
   Why: the composition root shows which handlers, provider bundles, and orchestration policies are actually live.
2. `internal/services/ai/api/grpc/ai/`
   Why: transport owns request validation, authorization, and protobuf mapping. Handlers are thin wrappers over service methods, including the typed `AgentAuthReference` protobuf mapping for agent workflows.
3. `internal/services/ai/service/`
   Why: use-case orchestration — auth token resolution, access control, usage readers/policy, and audit. This is where business logic lives.
4. `internal/services/ai/agent/`, `internal/services/ai/credential/`, `internal/services/ai/providergrant/`, `internal/services/ai/accessrequest/`, and `internal/services/ai/providerconnect/`
   Why: the durable lifecycle and support-workflow rules now live in owning packages, including typed auth references, provider-grant refresh transitions, and provider OAuth connect-session state.
5. `internal/services/ai/orchestration/`, `internal/services/ai/orchestration/gametools/`, `internal/services/ai/orchestration/daggerhearttools/`, `internal/services/ai/orchestration/daggerheart/`, `internal/services/ai/providercatalog/`, `internal/services/ai/provider/openai/`, `internal/services/ai/provider/anthropic/`, and `internal/services/ai/provideroauth/`
   Why: campaign-turn execution is split between orchestration-owned prompt/runtime policy, the centralized production tool registry and direct-session shell, extracted Daggerheart dice/mechanics executors, current Daggerheart-specific prompt context sources, runtime provider bundle registration, provider-specific HTTP/model behavior, and the shared OAuth handshake capability contracts used by provider-grant runtime code. Anthropic currently stops at direct invocation/model listing; OpenAI is still the only provider with OAuth and tool-runtime support.
6. `internal/services/ai/campaigncontext/` plus `instructionset/`, `memorydoc/`, and `referencecorpus/`
   Why: artifact defaults, instruction loading, writable memory structure, and read-only reference corpus logic are now separate packages with different ownership.
7. `internal/services/ai/storage/` and `internal/services/ai/storage/sqlite/`
   Why: aggregate repository seams still live in `storage/`, while concrete durability lives in `storage/sqlite/`; support workflow stores such as provider connect, debug traces, audit events, and campaign artifacts now stay with their owning packages.

## Where to add tests

| If you changed... | Put tests here first | Why |
| --- | --- | --- |
| Domain/provider identity or status behavior | package-local `*_test.go` next to the owning package under `internal/services/ai/` | Durable invariants belong with the owning package, not in transport regression hubs. |
| Service-layer use-case logic, auth resolution, or access control | `internal/services/ai/service/*_test.go` | Business logic tests belong with the service layer, not in transport handler tests. |
| gRPC request validation, authz, or protobuf mapping | handler-family tests under `internal/services/ai/api/grpc/ai/` and any adjacent `*_test_support_test.go` file for that family | Transport seams should assert behavior where the public contract is assembled without rebuilding one package-wide regression framework. |
| Provider adapter request/response translation | `internal/services/ai/provider/openai/*_test.go`, `internal/services/ai/provider/anthropic/*_test.go`, and package-local `provider/*_test.go` | Keep provider HTTP behavior outside transport tests and assert shared provider vocabulary at the provider seam directly. |
| Campaign-turn loop, prompt assembly, timeout, or usage aggregation | `internal/services/ai/orchestration/*_test.go` | Orchestration has its own seam and should carry its own runtime coverage. |
| SQLite storage behavior | `internal/services/ai/storage/sqlite/*_test.go` | SQL and persistence invariants belong with the concrete adapter. |
| AI package-local fakes or spies | package-local `*_test.go` or `*_test_support_test.go` first; promote only genuinely shared seams to `internal/test/mock/aifakes/*` | Prefer capability-specific local fakes unless multiple packages truly share the same repository seam. |
| Cross-service AI GM flows | `internal/test/integration/` and worker-domain tests | Use integration only when game, AI, and worker behavior must line up. |

## Verification

- `go test ./internal/services/ai/...`
- `make test`
- `make check`

Add these when applicable:

- `make proto` for `api/proto/ai/v1/service.proto` changes
- `go test -tags=integration ./internal/test/integration -run 'TestAIDirectSessionDaggerheart(CombatFlowTools|MechanicsTools)|TestAIGMCampaignContextReplay'` for deterministic AI orchestration, replay, or direct-session tool contract changes
- `INTEGRATION_OPENAI_API_KEY=... go test -tags='integration liveai' ./internal/test/integration -run TestAIGMCampaignContextLiveCaptureBootstrap -count=1` for live-model refresh of the bootstrap lane after tool or instruction changes

For AI-specific live-capture lane selection beyond bootstrap, use the canonical
[Integration tests](../running/integration-tests.md) guide rather than copying
the full lane list into this contributor map.

## Related docs

- [AI service architecture](../architecture/platform/ai-service-architecture.md)
- [AI service lifecycle terms](ai-service-lifecycle-terms.md)
- [Campaign AI agent system](../architecture/platform/campaign-ai-agent-system.md)
- [Campaign AI mechanics quality](../architecture/platform/campaign-ai-mechanics-quality.md)
- [Campaign AI orchestration](../architecture/platform/campaign-ai-orchestration.md)
- [Campaign AI session bootstrap](../architecture/platform/campaign-ai-session-bootstrap.md)
- [Small services topology](small-services-topology.md)
