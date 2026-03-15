---
title: "Web testing map"
parent: "Platform surfaces"
nav_order: 13
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# Web Testing Map

Quick orientation for contributors choosing where web changes should be tested.

Canonical implementation path: `internal/services/web/`.

## Start Here

- Run `make test` during implementation.
- Run `make web-architecture-check` when changing web boundaries, routes,
  modules, or templates.
- Run `make smoke` when a change affects runtime composition, request flow, or
  browser-visible behavior that needs quick end-to-end confidence.
- Run `make check` before push, PR open, or PR update.

## High-Signal Root Web Coverage

- `internal/services/web/server_test.go`
  Root protected/public handler behavior, redirects, nav chrome, and mounted
  module interactions.
- `internal/services/web/server_locale_test.go`
  Request locale resolution, shell language, and localized error/page paths.
- `internal/services/web/server_viewer_test.go`
  Signed-in viewer shell behavior such as menus, avatar/profile links, and
  notification affordances.
- `internal/services/web/server_static_test.go`
  Static/public asset and shell contract coverage.
- `internal/services/web/server_test_harness_defaults_test.go`
  Default dependency bundle shapes used by root web tests.
- `internal/services/web/server_test_harness_helpers_test.go`
  Explicit test dependency completion rules and cross-module harness helpers.
- `internal/cmd/web/web_test.go`
  Startup dependency policy, dependency bootstrapping, and command/runtime
  wiring.

## High-Signal Shared-Platform Coverage

- `internal/services/web/principal/requeststate_test.go`
  Shared request-state contracts used by handler bases, page rendering, and
  error helpers.
- `internal/services/web/platform/modulehandler/modulehandler_test.go`
  Protected handler-base behavior for user-id propagation, localization, and
  page/error rendering helpers.
- `internal/services/web/platform/publichandler/publichandler_test.go`
  Public handler-base behavior for signed-in branching, localization, and page
  shell helpers.
- `internal/services/web/platform/pagerender/pagerender_test.go`
  Shared app-shell/public-shell page writing behavior once page state is
  resolved.
- `internal/services/web/platform/weberror/weberror_test.go`
  Shared protected/public error rendering behavior and safe message handling.
- `internal/services/web/platform/dashboardsync/sync_test.go`
  Shared dashboard freshness invalidation and degraded-mode behavior.

## High-Signal Module Coverage

- `internal/services/web/modules/<area>/routes_test.go`
  Route contract and method coverage for the owning area.
- `internal/services/web/modules/<area>/handlers*_test.go`
  Area-owned transport behavior when a route has meaningful branching or page
  assembly.
- `internal/services/web/modules/architecture_test.go`
  Cross-module AST-based architecture rules and module-template invariants.
- `internal/services/web/modules/boundary_guardrails_test.go`
  Cross-module boundary guardrails for known hotspot seams and ownership cuts.
  Prefer adding AST/package-construction invariants there over raw source-text
  fragment scans when refactors move code around without changing ownership.

## Where To Add Tests

- Startup wiring or backend dependency policy:
  `internal/cmd/web/web_test.go`
- Root request/principal/public shell behavior:
  `internal/services/web/server_test.go`
- Locale, viewer, or shell-level chrome behavior:
  `internal/services/web/server_locale_test.go`
  `internal/services/web/server_viewer_test.go`
- Shared request-state or transport helper behavior:
  `internal/services/web/principal/requeststate_test.go`
  `internal/services/web/platform/modulehandler/modulehandler_test.go`
  `internal/services/web/platform/publichandler/publichandler_test.go`
  `internal/services/web/platform/pagerender/pagerender_test.go`
  `internal/services/web/platform/weberror/weberror_test.go`
  `internal/services/web/platform/dashboardsync/sync_test.go`
- Module route or handler behavior:
  the owning module `routes_test.go` and any owned `handlers*_test.go`
- Shared web architecture or boundary rules:
  `internal/services/web/modules/architecture_test.go`
  `internal/services/web/modules/boundary_guardrails_test.go`

## Contributor Rule

When a refactor changes the canonical place to verify a web behavior, update
this map in the same slice. Contributors should not need to rediscover the
test entrypoints by reading the whole package tree.
