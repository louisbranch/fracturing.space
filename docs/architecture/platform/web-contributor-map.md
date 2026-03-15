---
title: "Web contributor map"
parent: "Platform surfaces"
nav_order: 11
status: canonical
owner: engineering
last_reviewed: "2026-03-09"
---

# Web Contributor Map

Quick orientation for contributors changing the browser-facing web service.

Canonical implementation path: `internal/services/web/`.

## Start Here

- Route ownership starts in `routepath/`, module `routes.go`, and module `module.go`.
  Keep `routepath` edits inside the owned surface file for the area you are
  changing instead of growing a cross-area route constant bucket again.
- Protected module request flow is usually `routes.go` -> `handlers*.go` -> `app/` -> `gateway/`.
- Public module request flow is usually `routes.go` -> `handlers.go` -> `app/` -> `gateway/`, with `publichandler.Base` for shared rendering behavior.
- Top-level startup/composition lives outside feature areas:
  - `cmd/web`
  - `internal/cmd/web`
  - `internal/services/web/{server.go,principal/,composition/,app/,modules/}`
- Startup dependency policy lives in `internal/cmd/web/dependency_graph.go`.
  Read that file before changing whether a backend outage should block startup
  or only degrade specific web surfaces.
- Startup dependency assembly lives in `internal/cmd/web/runtime_dependencies.go`.
  Command-layer code owns addresses, dependency policy, and connection
  lifecycle there. Service-owned bundle construction and client binding live in
  `internal/services/web/dependencies.go`; do not patch partially-built bundles
  later in `Run`.

## Package Roles

- `internal/services/web/principal`: request-scoped session validation, viewer chrome, locale resolution, and the middleware-owned principal snapshot.
  Request-time signed-in state belongs here; public modules should consume
  `ResolveSignedIn` instead of re-validating cookies through feature gateways.
- `internal/services/web/platform/requestresolver`: shared request-scoped
  viewer/language contract used by `modulehandler`, `publichandler`,
  `pagerender`, and `weberror`, plus the grouped principal contract threaded
  through `server`, `composition`, and `modules`. It also owns the shared
  localized page-state helpers used by shell rendering, error responses, and
  direct public-page localization.
  Start here when changing app-shell request resolution flow.
- `internal/services/web/module`: canonical module contract types only.
- `internal/services/web/composition`: turns resolved principal callbacks and module dependencies into the app handler.
- `internal/services/web/app`: root mux composition, auth wrapping, and same-origin protections.
- `internal/services/web/modules`: registry builder plus module dependency bundles; it should not re-export the singular `module` contract, and registry files should call area-owned `Compose(...)` entrypoints instead of constructing feature gateways inline.
  When one area owns multiple mounted surfaces, prefer an area-owned
  `Compose...Set(...)` entrypoint there instead of repeating the surface list in
  the registry.
- `internal/services/web/modules/<area>`: route owner for one area.
- `internal/services/web/modules/<area>/app`: area-local orchestration and input validation.
- `internal/services/web/modules/<area>/gateway`: backend protocol mapping.
- `internal/services/web/modules/<area>/render`: area-owned render/view-model
  seams when a page set has outgrown shared `templates/`.
- `internal/services/web/modules/<area>/workflow`: transport-owned
  system-specific workflow contracts and implementations when one area has
  multiple workflow adapters.
- `internal/services/web/modules/notifications`: inbox transport/view mapping
  stays area-owned, but the canonical notification message payload contract
  lives in `internal/services/shared/notificationpayload`.
- `internal/services/web/platform/*`: shared transport helpers only; feature-specific behavior should not accumulate here.
- `internal/services/web/templates`: shared layout and templ primitives. If one
  area's page set becomes a contributor hotspot, move that set under the
  owning area instead of extending the shared package indefinitely.

## Where Changes Usually Belong

- New route or changed route contract: owning module `routes.go`, `module.go`,
  and the matching owned file in `routepath/`.
- Changed page behavior with the same backend shape: owning module
  handlers/view mapping first, then the area-owned render seam if one exists,
  and shared `templates` only for shell-level primitives.
- Changed web-only workflow or validation: owning module `app/`.
- Changed backend transport mapping or proto normalization: owning module `gateway/`.
- Shared auth, request, session, or page shell behavior: `internal/services/web/principal`, `internal/services/web/platform/`, or root composition packages, but only after confirming it is truly cross-cutting.

## Current Hotspots

- `campaigns`: still the largest area; the root gateway/service shims,
  duplicate app-owned workflow registry, root-owned workflow contract,
  production alias wall, and broad flat gateway client bag are gone, and the
  campaign detail handlers and module-owned markup now live under
  `campaigns/render`, while list/start/create/chat pages live under the root
  `campaigns` package. Character-creation assembly and step parsing now live in
  `campaigns/workflow/service.go`, and production module wiring now passes
  read/mutation/authz seams explicitly, including test module construction.
  The remaining contributor caution is mostly app/gateway size and route-surface
  complexity, not shared page ownership or broad constructor bags.
- `settings`: route/files already split by account and AI ownership, and the
  constructor seam now matches that split via explicit account-vs-AI app
  services in `settings/module.go` and `settings/app/service.go`.
- `publicauth`: page templates are module-owned, and post-auth continuation
  path validation is shared via `publicauth/redirectpath` instead of being
  duplicated in handlers and the app service. Request-time signed-in detection
  now comes from `principal.ResolveSignedIn`; `publicauth` owns auth ceremonies
  and logout/session revocation, not duplicate session-validation policy. Its
  multi-surface public module set is composed inside `publicauth` rather than
  listed inline in the central registry.
- `templates`: now functions as a shared shell/layout primitive package. Keep
  area-owned pages out of it instead of re-growing a cross-area page bucket.

## Guardrails To Trust

- `internal/services/web/modules/architecture_test.go`
- `internal/services/web/modules/boundary_guardrails_test.go`
- `internal/services/web/modules/constructor_guardrails_test.go`
- `internal/services/web/routepath_guardrails_test.go`
- `internal/services/web/templates/routes_guardrails_test.go`
- per-module `routes_test.go` files

When changing boundaries, update docs and guardrails in the same slice so the
next contributor inherits the new shape rather than reverse-engineering it.
