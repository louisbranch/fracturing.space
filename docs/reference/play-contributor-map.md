---
title: "Play contributor map"
parent: "Reference"
nav_order: 16
status: canonical
owner: engineering
last_reviewed: "2026-03-19"
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
| Change browser payload contracts or websocket frame shapes | `internal/services/play/protocol/` |
| Change shell routes or browser handoff/session flow | `internal/services/play/app/shell_transport.go`, `session.go`, `request_context.go`, `shell.go` |
| Change bootstrap/history/realtime response assembly | `internal/services/play/app/application.go` |
| Change bootstrap/history API request mapping | `internal/services/play/app/api_transport.go`, `request_context.go` |
| Change the overall browser route surface or top-level route registration | `internal/services/play/app/routes.go`, `interaction_routes.go` |
| Change interaction mutation routing or shared mutation transport flow | `internal/services/play/app/interaction_routes.go`, `interaction_transport.go`, `request_context.go` |
| Change websocket framing, room lifecycle, typing, or fanout | `internal/services/play/app/realtime_*.go` |
| Change transcript contracts, validation, or pagination defaults | `internal/services/play/transcript/` |
| Change reusable transcript adapter contract tests | `internal/services/play/transcript/transcripttest/` |
| Change SQLite transcript behavior, retries, or migrations | `internal/services/play/storage/sqlite/` |
| Change the bundled play shell placeholder, injected shell config, or path resolution | `internal/services/play/ui/src/App.tsx`, `app_mode.ts`, `shell_config.ts`, `internal/services/play/app/shell.go` |
| Change Storybook-first component workflow guidance | `docs/guides/play-ui-component-preview-workflow.md`, `internal/services/play/ui/src/storybook/` |
| Change shared HUD/component fixtures, interaction view-model mapping, or component contracts | `internal/services/play/ui/src/interaction/player-hud/shared/view-models.ts`, `internal/services/play/ui/src/interaction/` |
| Change system-specific presentation | `internal/services/play/ui/src/systems/` |

## Package reading order

1. `internal/cmd/play/`
   Why: composition root shows what is built and owned outside the runtime.
2. `internal/services/play/app/`
   Why: browser transport and active-play runtime ownership live here, split by request-context resolution, a full route catalog, shell flow, API request mapping, application assembly, mutation transport, and realtime runtime.
3. `internal/services/play/protocol/`
   Why: browser contract ownership lives here as play-owned DTOs instead of being spread across handlers or leaked directly from generated gameplay structs.
4. `internal/services/play/transcript/`
   Why: transcript contract defines scope, append, and history query invariants.
5. `internal/services/play/storage/sqlite/`
   Why: concrete persistence behavior, ordering/idempotency, and concurrent retry behavior live here.
6. `internal/services/play/ui/src/`
   Why: the bundled placeholder shell, Storybook-first component slices, shared fixtures, and renderer extension seams live here.

## Where to add tests

| If you changed... | Put tests here first | Why |
| --- | --- | --- |
| Runtime boundary or constructor ownership | `internal/services/play/app/architecture_test.go` and focused package tests | Protect app-vs-command ownership and runtime seams. |
| Browser contract or websocket payload shape | `internal/services/play/protocol/` tests plus focused `internal/services/play/ui/src/**/*.test.ts*` tests when the UI consumes the payloads | Keep the transport contract explicit without inventing ad hoc UI transport types. |
| HTTP/session/bootstrap behavior | `internal/services/play/app/*_test.go` | Keep browser transport assertions at the transport seam and application refresh flow. |
| Request-context resolution or route inventory | `internal/services/play/app/api_transport_test.go`, `shell_transport_test.go`, `interaction_routes_test.go`, and `routes_test.go` | Keep campaign/auth parsing plus the full indexed browser route surface explicit for contributors. |
| Realtime behavior | focused `internal/services/play/app/realtime*_test.go` tests | Room, websocket, timer, and retry behavior are runtime-package concerns and should stay near the owning runtime files. |
| Transcript contract defaults or request/query validation | `internal/services/play/transcript/*_test.go` | Keep the canonical store seam explicit outside any one adapter. |
| SQLite transcript behavior | `internal/services/play/storage/sqlite/*_test.go` plus `internal/services/play/transcript/transcripttest/` | Ordering, idempotency, and concurrent retry behavior belong with the adapter while the reusable contract stays reader-visible. |
| Placeholder shell path behavior | `internal/services/play/ui/src/App.test.tsx` and `app_mode.test.ts` | Keep the shipped play shell surface explicit while the runtime remains a placeholder. |
| Storybook/component behavior | the nearest `*.test.tsx` beside the component slice | Keep component contracts, fixtures, and isolated stories aligned without depending on the placeholder shell. |
| Renderer-facing labels or HUD view-model normalization for display | `internal/services/play/ui/src/interaction/player-hud/shared/view-models.test.ts` plus renderer tests under `internal/services/play/ui/src/systems/` | Keep display mapping and fallback labels out of unrelated components. |

## Verification

- `go test ./internal/services/play/... ./internal/cmd/play/...`
- `go test -race ./internal/services/play/app ./internal/services/play/storage/sqlite`
- `make play-architecture-check`
- `make play-ui-check` when changing `internal/services/play/ui/**`
- `make smoke` for browser/runtime path changes
- `make check` before push or PR update

## Related docs

- [Play architecture](../architecture/platform/play-architecture.md)
- [Interaction surfaces](../architecture/platform/interaction-surfaces.md)
- [Small services topology](small-services-topology.md)
