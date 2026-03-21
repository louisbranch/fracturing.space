---
title: "AI Service Architecture"
parent: "Platform surfaces"
nav_order: 16
status: canonical
owner: engineering
last_reviewed: "2026-03-20"
---

# AI Service Architecture

High-level overview of the AI service's internal structure. For behavioral
details see the companion docs:

- [Campaign AI orchestration](campaign-ai-orchestration.md) — grant, tool
  policy, turn-loop mechanics
- [Campaign AI agent system](campaign-ai-agent-system.md) — instruction
  composition, context assembly, extension points
- [Campaign AI session bootstrap](campaign-ai-session-bootstrap.md) — bootstrap
  turn behavior and future improvements
- [AI service contributor map](../../reference/ai-service-contributor-map.md) —
  package routing for contributors

## Layer Diagram

```
┌─────────────────────────────────────────────────────┐
│                    Transport                         │
│  api/grpc/ai/  — proto parse → service → proto resp │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────▼──────────────────────────────┐
│                  Service Layer                       │
│  service/  — use-case orchestration, auth token      │
│  resolution, access control, usage guards            │
└──────┬───────────────┬──────────────────────────────┘
       │               │
┌──────▼──────┐ ┌──────▼──────────────────────────────┐
│   Domain    │ │           Orchestration              │
│  agent/     │ │  orchestration/  — turn runner,      │
│  credential/│ │  prompt builder, context sources      │
│  provider-  │ │  orchestration/gametools/  — tool     │
│   grant/    │ │  execution, gRPC session management   │
│  access-    │ │  orchestration/daggerheart/  — game-  │
│   request/  │ │  system-specific context sources      │
└──────┬──────┘ └──────┬──────────────────────────────┘
       │               │
┌──────▼───────────────▼──────────────────────────────┐
│                    Storage                           │
│  storage/  — interface contracts (domain types)      │
│  storage/sqlite/  — concrete SQLite adapter          │
└─────────────────────────────────────────────────────┘
```

Supporting packages sit alongside these layers:

- `provider/` and `provider/openai/` — provider identity, OAuth, model listing,
  invocation, and Responses API translation
- `campaigncontext/` — artifact defaults, instruction loading, memory document
  structure, reference corpus
- `secret/` — encryption seam for credential and grant secrets
- `app/` — composition root wiring all layers into the live process

## Domain Models Own the Truth

Each domain package (`agent`, `credential`, `providergrant`, `accessrequest`)
defines the canonical type used by all layers. There are no separate "storage
record" types — the SQLite adapter scans directly into domain structs. Typed
fields (e.g., `accessrequest.Status`, `providergrant.Status`, `agent.Status`,
`provider.Provider`) replace raw strings throughout.

Lifecycle transitions (`Create`, `Review`, `Revoke`, `Refresh`) live in the
owning domain package as pure functions. The service layer calls these functions,
then persists the result.

## Service Layer

`internal/services/ai/service/` contains one service struct per workflow family:

| Service | Responsibility |
|---------|---------------|
| `AgentService` | Agent CRUD, auth state, model listing, campaign binding validation |
| `CredentialService` | Credential creation (with encryption), listing, revocation |
| `ProviderGrantService` | OAuth connect flow (PKCE), grant listing, revocation |
| `AccessRequestService` | Shared-access request lifecycle and audit |
| `InvocationService` | Single-agent invocation with access control |
| `CampaignOrchestrationService` | Campaign turn execution with grant validation |

Supporting infrastructure in the same package:

- `AuthTokenResolver` — resolves agent auth references to live provider tokens
- `AccessibleAgentResolver` — ownership or approved-access checks
- `UsageGuard` — blocks mutations to resources bound to active campaigns
- `Error`/`ErrorKind` — typed service errors mapped to gRPC codes by transport

## Transport Layer

The transport package is a thin gRPC wrapper. Each handler method:
1. Extracts the caller identity from gRPC metadata
2. Parses the proto request into service-layer inputs
3. Calls the corresponding service method
4. Converts the domain result to a proto response

No business logic lives in the transport layer. Error translation from
`service.ErrorKind` to gRPC status codes happens in a central mapping function.

## Authentication Model

Agents reference their provider authentication via `AuthReference`, a sealed
sum type with two variants:

- **Credential** — a stored encrypted API key (`credential.Credential`)
- **Provider Grant** — an OAuth token pair (`providergrant.ProviderGrant`)

The `AuthTokenResolver` in the service layer resolves the reference to a live
token at invocation time, handling credential decryption and provider-grant
refresh transparently.

## How to Add a New RPC

1. Define the RPC in `api/proto/ai/v1/service.proto` and run `make proto`.
2. Add a service method in the appropriate `service/*.go` file.
3. Add a thin handler method in the transport package that parses the proto,
   calls the service, and converts the result.
4. Register the handler in the appropriate handler root file.
5. Add handler-level tests in the transport package test files.

## How to Add a Game System Tool

1. Define the tool in `orchestration/gametools/tools_*.go`.
2. Implement the tool's execution logic using `gametools.Session` gRPC calls.
3. Register the tool in the tool profile returned by `gametools.Tools()`.
4. Add game-system-specific context sources in
   `orchestration/{system}/` and register them in the composition root.
5. Add instruction files under `data/instructions/v1/{system}/`.
