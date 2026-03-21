---
title: "AI service contributor map"
parent: "Reference"
nav_order: 17
status: canonical
owner: engineering
last_reviewed: "2026-03-18"
---

# AI service contributor map

Reader-first routing guide for contributors changing the AI service.

## Start here

Read in this order:

1. [Architecture foundations](../architecture/foundations/index.md)
2. [Campaign AI orchestration](../architecture/platform/campaign-ai-orchestration.md)
3. [Campaign AI session bootstrap](../architecture/platform/campaign-ai-session-bootstrap.md)
4. [Campaign AI agent system](../architecture/platform/campaign-ai-agent-system.md)
5. [AI service lifecycle terms](ai-service-lifecycle-terms.md)
6. This page

Use [Verification commands](../running/verification.md) for the canonical local
check sequence.

## Where to edit

| Change you want | Primary packages/files |
| --- | --- |
| Change process startup, runtime config, or service registration | `internal/cmd/ai/`, `internal/services/ai/app/` |
| Change gRPC handler seams, auth checks, or proto mapping | `internal/services/ai/api/grpc/ai/` |
| Change use-case orchestration, auth token resolution, access control, or usage guards | `internal/services/ai/service/` |
| Change agent, credential, provider-grant, or access-request domain rules | `internal/services/ai/agent/`, `internal/services/ai/credential/`, `internal/services/ai/providergrant/`, `internal/services/ai/accessrequest/` |
| Change provider identity or provider-reported usage contracts | `internal/services/ai/provider/` |
| Change OpenAI OAuth, invocation, model listing, Responses API translation, or strict-schema policy | `internal/services/ai/provider/openai/` |
| Change campaign-turn orchestration, prompt assembly, tool dispatch, or provider step aggregation | `internal/services/ai/orchestration/`, `internal/services/ai/orchestration/gametools/` |
| Change campaign artifact bootstrapping or artifact path policy | `internal/services/ai/campaigncontext/`, `internal/services/ai/api/grpc/ai/*artifact*` |
| Change AI instruction-file loading, memory document structure, or system reference corpus logic | `internal/services/ai/campaigncontext/instructionset/`, `internal/services/ai/campaigncontext/memorydoc/`, `internal/services/ai/campaigncontext/referencecorpus/`, `internal/services/ai/api/grpc/ai/*reference*` |
| Change auth-reference typing, provider-grant refresh transitions, or lifecycle vocabulary | `internal/services/ai/agent/`, `internal/services/ai/providergrant/`, [ai-service-lifecycle-terms.md](ai-service-lifecycle-terms.md) |
| Change SQLite persistence or migration-backed storage contracts | `internal/services/ai/storage/sqlite/`, `internal/services/ai/storage/` |
| Add AI test doubles or handler test setup | `internal/test/mock/aifakes/`, `internal/services/ai/api/grpc/ai/transport_test_helpers_test.go` |
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
   Why: the composition root shows which handlers, provider adapters, and orchestration policies are actually live.
2. `internal/services/ai/api/grpc/ai/`
   Why: transport owns request validation, authorization, and protobuf mapping. Handlers are thin wrappers over service methods.
3. `internal/services/ai/service/`
   Why: use-case orchestration — auth token resolution, access control, usage guards, and audit. This is where business logic lives.
4. `internal/services/ai/agent/`, `internal/services/ai/credential/`, `internal/services/ai/providergrant/`, and `internal/services/ai/accessrequest/`
   Why: the durable lifecycle rules now live in the owning domain packages, including typed auth references and provider-grant refresh transitions.
5. `internal/services/ai/orchestration/`, `internal/services/ai/orchestration/gametools/`, and `internal/services/ai/provider/openai/`
   Why: campaign-turn execution is split between orchestration-owned prompt/tool/runtime policy, concrete game-tool execution, and provider-specific HTTP/OAuth/model behavior outside the transport package.
6. `internal/services/ai/campaigncontext/` plus `instructionset/`, `memorydoc/`, and `referencecorpus/`
   Why: artifact defaults, instruction loading, writable memory structure, and read-only reference corpus logic are now separate packages with different ownership.
7. `internal/services/ai/storage/` and `internal/services/ai/storage/sqlite/`
   Why: auth-reference persistence, refresh state, and artifact durability live here, with sqlite coverage now split by aggregate instead of one omnibus test file.

## Where to add tests

| If you changed... | Put tests here first | Why |
| --- | --- | --- |
| Domain/provider identity or status behavior | package-local `*_test.go` next to the owning package under `internal/services/ai/` | Durable invariants belong with the owning package, not in transport regression hubs. |
| Service-layer use-case logic, auth resolution, or access control | `internal/services/ai/service/*_test.go` | Business logic tests belong with the service layer, not in transport handler tests. |
| gRPC request validation, authz, or protobuf mapping | handler-family tests under `internal/services/ai/api/grpc/ai/` plus `transport_test_helpers_test.go` | Transport seams should assert behavior where the public contract is assembled without collapsing every RPC family into one regression hub. |
| Provider adapter request/response translation | `internal/services/ai/provider/openai/*_test.go` and package-local `provider/*_test.go` | Keep provider HTTP/Responses API behavior outside transport tests and assert shared provider vocabulary at the provider seam directly. |
| Campaign-turn loop, prompt assembly, timeout, or usage aggregation | `internal/services/ai/orchestration/*_test.go` | Orchestration has its own seam and should carry its own runtime coverage. |
| SQLite storage behavior | `internal/services/ai/storage/sqlite/*_test.go` | SQL and persistence invariants belong with the concrete adapter. |
| AI package-local fakes or spies | `internal/test/mock/aifakes/*` | Prefer capability-specific fakes over a new omnibus fake so tests only depend on the repository surface they actually use. |
| Cross-service AI GM flows | `internal/test/integration/` and worker-domain tests | Use integration only when game, AI, and worker behavior must line up. |

## Verification

- `go test ./internal/services/ai/...`
- `make test`
- `make check`

Add these when applicable:

- `make proto` for `api/proto/ai/v1/service.proto` changes
- `go test ./internal/test/integration -tags=integration -run 'TestAIGM|TestGameEndToEnd'` for AI orchestration contract changes

## Related docs

- [AI service architecture](../architecture/platform/ai-service-architecture.md)
- [AI service lifecycle terms](ai-service-lifecycle-terms.md)
- [Campaign AI agent system](../architecture/platform/campaign-ai-agent-system.md)
- [Campaign AI orchestration](../architecture/platform/campaign-ai-orchestration.md)
- [Campaign AI session bootstrap](../architecture/platform/campaign-ai-session-bootstrap.md)
- [Small services topology](small-services-topology.md)
