---
title: "AI service contributor map"
parent: "Reference"
nav_order: 17
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# AI service contributor map

Reader-first routing guide for contributors changing the AI service.

## Start here

Read in this order:

1. [Architecture foundations](../architecture/foundations/index.md)
2. [Campaign AI orchestration](../architecture/platform/campaign-ai-orchestration.md)
3. [Campaign AI session bootstrap](../architecture/platform/campaign-ai-session-bootstrap.md)
4. This page

Use [Verification commands](../running/verification.md) for the canonical local
check sequence.

## Where to edit

| Change you want | Primary packages/files |
| --- | --- |
| Change process startup, runtime config, or service registration | `internal/cmd/ai/`, `internal/services/ai/app/` |
| Change gRPC handler seams, auth checks, or proto mapping | `internal/services/ai/api/grpc/ai/` |
| Change agent, credential, provider-grant, or access-request domain rules | `internal/services/ai/agent/`, `internal/services/ai/credential/`, `internal/services/ai/providergrant/`, `internal/services/ai/accessrequest/` |
| Change provider identity or provider-reported usage contracts | `internal/services/ai/provider/` |
| Change OpenAI OAuth, invocation, model listing, or Responses API adapters | `internal/services/ai/provider/openai/` |
| Change campaign-turn orchestration, prompt assembly, tool dispatch, or provider step aggregation | `internal/services/ai/orchestration/`, `internal/services/ai/orchestration/gametools/` |
| Change campaign artifact bootstrapping or reference lookup logic | `internal/services/ai/campaigncontext/`, `internal/services/ai/api/grpc/ai/*artifact*`, `internal/services/ai/api/grpc/ai/*reference*` |
| Change SQLite persistence or migration-backed storage contracts | `internal/services/ai/storage/sqlite/`, `internal/services/ai/storage/` |
| Change cross-service campaign-turn or replay/integration coverage | `internal/test/integration/`, `internal/services/worker/domain/` |

## Package reading order

1. `internal/services/ai/app/`
   Why: the composition root shows which handlers, provider adapters, and orchestration policies are actually live.
2. `internal/services/ai/api/grpc/ai/`
   Why: transport owns request validation, authorization, and protobuf mapping into the internal seams.
3. `internal/services/ai/orchestration/` and `internal/services/ai/provider/openai/`
   Why: campaign-turn execution and provider-specific behavior are intentionally outside the transport package.
4. `internal/services/ai/storage/` and `internal/services/ai/storage/sqlite/`
   Why: auth-reference persistence, refresh state, and artifact durability live here.

## Where to add tests

| If you changed... | Put tests here first | Why |
| --- | --- | --- |
| Domain/provider identity or status behavior | package-local `*_test.go` next to the owning package under `internal/services/ai/` | Durable invariants belong with the owning package, not in transport regression hubs. |
| gRPC request validation, authz, or protobuf mapping | `internal/services/ai/api/grpc/ai/service_test.go` and focused handler tests | Transport seams should assert behavior where the public contract is assembled. |
| Provider adapter request/response translation | `internal/services/ai/provider/openai/*_test.go` | Keep provider HTTP/Responses API behavior outside transport tests. |
| Campaign-turn loop, prompt assembly, timeout, or usage aggregation | `internal/services/ai/orchestration/*_test.go` | Orchestration has its own seam and should carry its own runtime coverage. |
| SQLite storage behavior | `internal/services/ai/storage/sqlite/*_test.go` | SQL and persistence invariants belong with the concrete adapter. |
| Cross-service AI GM flows | `internal/test/integration/` and worker-domain tests | Use integration only when game, AI, and worker behavior must line up. |

## Verification

- `go test ./internal/services/ai/...`
- `make test`
- `make check`

Add these when applicable:

- `make proto` for `api/proto/ai/v1/service.proto` changes
- `go test ./internal/test/integration -tags=integration -run 'TestAIGM|TestGameEndToEnd'` for AI orchestration contract changes

## Related docs

- [Campaign AI orchestration](../architecture/platform/campaign-ai-orchestration.md)
- [Campaign AI session bootstrap](../architecture/platform/campaign-ai-session-bootstrap.md)
- [Small services topology](small-services-topology.md)
