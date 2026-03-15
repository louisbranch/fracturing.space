---
title: "Play contributor map"
parent: "Reference"
nav_order: 16
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# Play contributor map

Reader-first routing guide for contributors changing the play service.

## Start here

Read in this order:

1. [Architecture foundations](../architecture/foundations/index.md)
2. [Interaction surfaces](../architecture/platform/interaction-surfaces.md)
3. [Play architecture](../architecture/platform/play-architecture.md)
4. This page

Use [Verification commands](../running/verification.md) for the canonical local
check sequence.

## Where to edit

| Change you want | Primary packages/files |
| --- | --- |
| Change process startup, dependency wiring, or shutdown ordering | `internal/cmd/play/` |
| Change browser payload contracts or websocket frame shapes | `internal/services/play/protocol/`, `internal/services/play/ui/src/protocol.ts` |
| Change shell routes or browser handoff/session flow | `internal/services/play/app/shell_transport.go`, `session.go`, `request_context.go`, `shell.go` |
| Change bootstrap/history/realtime response assembly | `internal/services/play/app/application.go` |
| Change bootstrap/history API request mapping | `internal/services/play/app/api_transport.go`, `request_context.go` |
| Change the overall browser route surface or top-level route registration | `internal/services/play/app/routes.go`, `interaction_routes.go` |
| Change interaction mutation routing or shared mutation transport flow | `internal/services/play/app/interaction_routes.go`, `interaction_transport.go`, `request_context.go` |
| Change websocket framing, room lifecycle, typing, or fanout | `internal/services/play/app/realtime_*.go` |
| Change transcript contracts, validation, or pagination defaults | `internal/services/play/transcript/` |
| Change reusable transcript adapter contract tests | `internal/services/play/transcript/transcripttest/` |
| Change SQLite transcript behavior, retries, or migrations | `internal/services/play/storage/sqlite/` |
| Change SPA runtime state transitions | `internal/services/play/ui/src/runtime_state.ts` |
| Change SPA runtime fetch/websocket orchestration | `internal/services/play/ui/src/runtime.ts`, `runtime_transport.ts`, `realtime.ts` |
| Change renderer labels or UI-facing normalization of protocol state | `internal/services/play/ui/src/view_models.ts` |
| Change system-specific presentation | `internal/services/play/ui/src/systems/` |

## Package reading order

1. `internal/cmd/play/`
   Why: composition root shows what is built and owned outside the runtime.
2. `internal/services/play/app/`
   Why: browser transport and active-play runtime ownership live here, split by request-context resolution, a full route catalog, shell flow, API request mapping, application assembly, mutation transport, and realtime runtime.
3. `internal/services/play/protocol/`
   Why: browser contract ownership lives here instead of being spread across handlers and UI runtime code.
4. `internal/services/play/transcript/`
   Why: transcript contract defines scope, append, and history query invariants.
5. `internal/services/play/storage/sqlite/`
   Why: concrete persistence behavior, ordering/idempotency, and concurrent retry behavior live here.
6. `internal/services/play/ui/src/`
   Why: browser runtime state, transport clients, renderer view models, and renderer extension seams live here.

## Where to add tests

| If you changed... | Put tests here first | Why |
| --- | --- | --- |
| Runtime boundary or constructor ownership | `internal/services/play/app/architecture_test.go` and focused package tests | Protect app-vs-command ownership and runtime seams. |
| Browser contract or websocket payload shape | `internal/services/play/protocol/` tests plus `internal/services/play/ui/src/**/*.test.ts*` | Keep the transport contract explicit on both sides of the browser boundary. |
| HTTP/session/bootstrap behavior | `internal/services/play/app/*_test.go` | Keep browser transport assertions at the transport seam and application refresh flow. |
| Request-context resolution or route inventory | `internal/services/play/app/api_transport_test.go`, `shell_transport_test.go`, `interaction_routes_test.go`, and `routes_test.go` | Keep campaign/auth parsing plus the full indexed browser route surface explicit for contributors. |
| Realtime behavior | `internal/services/play/app/coverage_test.go` plus focused `realtime_*_test.go` tests | Room, websocket, timer, and retry behavior are runtime-package concerns. |
| Transcript contract defaults or request/query validation | `internal/services/play/transcript/*_test.go` | Keep the canonical store seam explicit outside any one adapter. |
| SQLite transcript behavior | `internal/services/play/storage/sqlite/*_test.go` plus `internal/services/play/transcript/transcripttest/` | Ordering, idempotency, and concurrent retry behavior belong with the adapter while the reusable contract stays reader-visible. |
| SPA runtime state behavior | `internal/services/play/ui/src/runtime_state.test.ts` and `internal/services/play/ui/src/runtime.test.tsx` | Keep reducer-like state transitions and hook orchestration tested separately. |
| Browser transport client behavior | `internal/services/play/ui/src/realtime.test.ts` and focused runtime hook tests | Websocket and fetch orchestration should stay explicit at the browser transport seam. |
| Renderer-facing labels or protocol normalization for display | `internal/services/play/ui/src/view_models.test.ts` and renderer tests under `internal/services/play/ui/src/systems/` | Keep numeric protocol interpretation and fallback labels out of components. |

## Verification

- `go test ./internal/services/play/... ./internal/cmd/play/...`
- `go test -race ./internal/services/play/app ./internal/services/play/storage/sqlite`
- `make play-architecture-check`
- `make smoke` for browser/runtime path changes
- `make check` before push or PR update

## Related docs

- [Play architecture](../architecture/platform/play-architecture.md)
- [Interaction surfaces](../architecture/platform/interaction-surfaces.md)
- [Small services topology](small-services-topology.md)
