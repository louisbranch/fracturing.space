---
title: "AI Service Architecture"
parent: "Platform surfaces"
nav_order: 16
status: canonical
owner: engineering
last_reviewed: "2026-03-23"
---

# AI Service Architecture

High-level overview of the AI service. For workflow details, use the companion
docs:

- [Campaign AI orchestration](campaign-ai-orchestration.md)
- [Campaign AI agent system](campaign-ai-agent-system.md)
- [Campaign AI session bootstrap](campaign-ai-session-bootstrap.md)
- [AI service contributor map](../../reference/ai-service-contributor-map.md)

## Layer Map

- Transport: `internal/services/ai/api/grpc/ai/`
  Thin proto parse/auth extraction/service call/proto response wrappers.
- Service: `internal/services/ai/service/`
  Workflow orchestration, access rules, auth-material resolution, provider
  capability checks, usage policy.
- Domain: `agent/`, `credential/`, `providergrant/`, `accessrequest/`
  Canonical lifecycle models and pure transition functions.
- Orchestration: `orchestration/`
  Turn runner, prompt builder, context sources, tool dispatch.
- Storage: `storage/`, `storage/sqlite/`
  Repository contracts plus the SQLite implementation.
- Composition root: `app/`
  Runtime wiring, provider registration, handler assembly, startup policy.

Supporting packages own stable shared vocabularies rather than leaving them in
`storage/`: `auditevent/`, `campaignartifact/`, `debugtrace/`,
`providerconnect/`, `provideroauth/`, and `gamebridge/`.

## Core Boundary Rules

- Domain packages own the truth. SQLite scans directly into domain or support
  package types rather than into storage-only DTOs.
- Lifecycle transitions stay pure in the owning package; services call them and
  persist the result.
- Interfaces are defined at consumption points. The app wiring composes live
  implementations but does not define business policy.
- Transport stays thin. Business logic belongs in `service/`, domain packages,
  provider adapters, or orchestration seams.
- Game collaboration crosses the AI-owned `gamebridge/` boundary rather than
  using raw game clients throughout the service tree.

## Service Responsibilities

| Service | Responsibility |
|---------|---------------|
| `AgentService` | Agent CRUD, binding validation, auth readiness, provider model listing |
| `CredentialService` | Encrypted credential create/list/revoke |
| `ProviderGrantService` | OAuth connect flow, grant list/revoke |
| `AccessRequestService` | Shared-access request lifecycle and audit |
| `InvocationService` | Direct single-agent invocation |
| `CampaignOrchestrationService` | Campaign-turn execution with grant validation |

Important service-local seams: `AuthMaterialResolver` resolves invoke-time auth
material, `ProviderGrantRuntime` owns grant refresh state, `ProviderConnectFinisher`
commits grant/session completion atomically, `AuthReferencePolicy` validates auth
usability and provider-model availability, and `UsagePolicy` turns usage reads into
mutation-blocking preconditions.

## Authentication and Provider Model

Agents bind to one `agent.AuthReference`, which is either a credential or a
provider grant. SQLite persists that as `auth_reference_type` plus
`auth_reference_id`, and the gRPC surface mirrors the same typed model through
`AgentAuthReference`.

Service code resolves an auth reference into invoke-time auth material. Past
that seam, provider-facing and orchestration-facing contracts intentionally use
the narrower `AuthToken` name because the domain distinction between decrypted
credential secret and refreshed access token has already been collapsed.

Provider capability registration is centralized in `providercatalog/`. The
composition root registers one bundle per implemented provider, and services ask
the registry for invocation, model listing, OAuth, or orchestration capability
instead of receiving parallel provider-specific maps.

Current runtime shape: OpenAI provides invocation, model listing, OAuth, and
campaign-turn orchestration support; Anthropic currently provides
credential-backed invocation and model listing only. Valid provider identity is
broader than current runtime availability, so services fail closed with
`FailedPrecondition` when a required capability is not registered.

## Orchestration Shape

`orchestration/` owns the generic turn runner, prompt builder, context-source
registry, and render pipeline. The runner now depends on an explicit
`TurnPolicy` seam for completion/reminder rules.

The context-source registry owns source naming, per-source tracing spans, and
typed brief merge rules. The always-on collector is exposed as
`NewCoreContextSourceRegistry()`, then extended by the composition root with
system-specific sources such as the Daggerheart sources in
`orchestration/daggerheart/`.

Tool execution is split:

- `orchestration/gametools/`
  Generic registry, direct-session shell, resource dispatch, shared catalogs
- `orchestration/daggerhearttools/`
  Daggerheart-specific mechanics, combat-flow, and read/resource execution

This is intentionally Daggerheart-first, not a generic per-system plugin
runtime yet.

## Transport and Tests

Transport handlers do four things only: extract caller identity, parse proto
requests, call one service method, and map results back to proto.

Tests are seam-first: package-local tests protect domain/provider/orchestration
invariants, service tests protect workflow behavior, and transport tests stay
limited to auth extraction, request validation, proto mapping, and handler-owned
response shaping.

## Extension Rules

To add a new RPC, update `api/proto/ai/v1/service.proto`, run `make proto`, add
the workflow method in the owning `service/*.go`, add a thin gRPC handler plus
app registration, and put behavioral coverage at the owning seam before adding
transport-only handler tests.

To add a new game-system tool, update the appropriate
`orchestration/gametools/tools_catalog_*.go`, keep system-specific execution in
the owning system package such as `orchestration/daggerhearttools/`, register
always-on prompt/context sources in the composition root, and update
instruction/reference content when the GM behavior contract changes.
