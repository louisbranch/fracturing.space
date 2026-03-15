---
title: "Web contributor map"
parent: "Platform surfaces"
nav_order: 11
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# Web Contributor Map

Quick orientation for contributors changing the browser-facing web service.

Canonical implementation path: `internal/services/web/`.

## Start Here

- Route ownership starts in `routepath/`, area `routes*.go`, and area `module.go`. Keep routepath edits inside the owned surface file for the area you are changing instead of regrowing a cross-area path bucket.
- Protected flow is usually `routes.go` -> `handlers*.go` -> `app/` -> `gateway/`.
- Public flow is usually `routes.go` -> `handlers.go` -> `app/` -> `gateway/`, with `publichandler.Base` for shared rendering behavior.
- Top-level startup and composition live outside feature areas: `cmd/web`, `internal/cmd/web`, and `internal/services/web/{server.go,principal/,composition/,app/,modules/}`.
- Start orientation with `doc.go` in `internal/services/web/`, `internal/services/web/module/`, and `internal/services/web/modules/` before dropping into implementation files.
- Startup dependency policy is defined in `internal/services/web/startup_dependencies.go`. Command-layer address mapping and connection lifecycle live in `internal/cmd/web/dependency_graph.go` and `internal/cmd/web/runtime_dependencies.go`.
- Service-owned dependency bundle construction lives in `internal/services/web/dependencies.go`; do not patch partially-built bundles later in `Run`.

## Package roles

- `internal/services/web/principal`: request-scoped session validation, viewer chrome, locale resolution, grouped principal callbacks, and the middleware-owned principal snapshot. Start here when changing app-shell request resolution flow.
- `internal/services/web/module`: canonical module contract types only. Shared request-state callback contracts belong in `principal`, not here.
- `internal/services/web/composition`: turns resolved principal callbacks and module dependencies into the app handler.
- `internal/services/web/app`: root mux composition, auth wrapping, and same-origin protections.
- `internal/services/web/modules`: registry builder plus module dependency bundles. Registry files call area-owned `Compose(...)` entrypoints instead of constructing feature gateways inline, and shared runtime helpers such as dashboard sync are built here once and passed into owning areas.
- `internal/services/web/modules/<area>`: route owner for one feature area.
- `internal/services/web/modules/<area>/app`: area-local orchestration and input validation.
- `internal/services/web/modules/<area>/gateway`: backend protocol mapping.
- For layered areas, the production flow is `composition.go` -> ready app service(s) in `Config` -> `module.go` -> `handlers*.go`. `Mount` should not rebuild app services from raw gateways.
- `internal/services/web/modules/<area>/render`: area-owned render and view-model seams once a page set outgrows shared `templates/`. Start with `doc.go`, then exported entrypoints, not generated `*_templ.go`.
- `internal/services/web/modules/<area>/workflow`: transport-owned system-specific workflow contracts and implementations when an area has multiple workflow adapters. Start with `doc.go`, then registry or service entrypoints, then system subpackages.
- `internal/services/web/modules/notifications`: inbox transport and view mapping stay area-owned, but the canonical notification payload contract lives in `internal/services/shared/notificationpayload`.
- `internal/services/web/platform/*`: shared transport helpers only. Start with package `doc.go` and package-local tests before editing those seams.
- `internal/services/web/templates`: shared shell and layout primitives. If one area's page set becomes a hotspot, move that set under the owning area instead of extending the shared package indefinitely.

For area-owned public surfaces that are optional by dependency, prefer `ComposePublic`-style constructors that return `(module.Module, bool)` from `composition.go` so the registry can explicitly include or omit whole routes based on configured clients.
For protected surfaces, prefer `ComposeProtected` constructors that centralize shared options and dependency mapping in `composition.go`, then gate optional protected surfaces (like notifications) in the registry based on explicit `configured` checks.

## Where changes usually belong

- New route or changed route contract: owning module `routes*.go`, `module.go`, and the matching owned file in `routepath/`.
- Changed page behavior with the same backend shape: owning module handlers and view mapping first, then the area-owned render seam if one exists, and shared `templates` only for shell-level primitives.
- Changed web-only workflow or validation: owning module `app/`.
- Changed backend transport mapping or proto normalization: owning module `gateway/`.
- Shared auth, request, session, or page shell behavior: `principal`, `platform/`, or root composition packages, but only after confirming it is truly cross-cutting.

## Current hotspots

- `campaigns`: still the largest area, but the root sink files are gone. Route registration, routepath ownership, app and gateway contracts, workflow registration, render entrypoints, and startup gateway deps are now split by owned surface. Start with `module.go`, the relevant `routes_*.go`, then `render/doc.go` or `workflow/doc.go` when those seams are involved.
- `settings`: route/files and production composition now keep account and AI ownership split end to end. Start with `composition.go`, then the matching account-vs-AI handler, app, and gateway files.
- `discovery`, `profile`, `invite`, `dashboard`, and `notifications`: all use the same small-module archetype where `composition.go` builds the production gateway plus app service and `module.go` only wires transport concerns.
- `publicauth`: continuation-path validation, signed-in detection, and page/session/passkey/recovery composition are all area-owned now. Start with `composition.go` and the specific capability files instead of looking for one transport-wide bundle.
- `templates`: shared shell/layout primitives only. Keep area-owned pages out of it.

## Guardrails to trust

- `internal/services/web/modules/architecture_test.go`
- `internal/services/web/modules/boundary_guardrails_test.go`
- `internal/services/web/modules/constructor_guardrails_test.go`
- `internal/services/web/routepath_guardrails_test.go`
- `internal/services/web/templates/routes_guardrails_test.go`
- Per-module `routes_test.go` files

The boundary guardrails prefer AST/package invariants and constructor contracts over brittle source-fragment policing, so harmless file reshapes should not force churn.

## High-signal coverage entrypoints

- Root runtime and shell behavior:
  `internal/services/web/server_test.go`
  `internal/services/web/server_locale_test.go`
  `internal/services/web/server_viewer_test.go`
  `internal/services/web/server_static_test.go`
- Root test harness ownership and explicit dependency completion:
  `internal/services/web/server_test_harness_defaults_test.go`
  `internal/services/web/server_test_harness_helpers_test.go`
- Startup dependency policy and runtime wiring:
  `internal/services/web/dependencies_test.go`
  `internal/cmd/web/web_test.go`
- Shared request-state and transport helpers:
  `internal/services/web/principal/requeststate_test.go`
  `internal/services/web/platform/modulehandler/modulehandler_test.go`
  `internal/services/web/platform/publichandler/publichandler_test.go`
  `internal/services/web/platform/pagerender/pagerender_test.go`
  `internal/services/web/platform/weberror/weberror_test.go`
  `internal/services/web/platform/dashboardsync/sync_test.go`
- Canonical test map:
  `docs/architecture/platform/web-testing-map.md`

When changing boundaries, update docs and guardrails in the same slice so the next contributor inherits the new shape instead of reverse-engineering it.
