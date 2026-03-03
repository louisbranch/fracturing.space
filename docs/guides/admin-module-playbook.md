---
title: "Admin module playbook"
parent: "Guides"
nav_order: 5
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Admin Module Playbook

Default way to add or modify an admin area module.

Canonical implementation path: `internal/services/admin/`.

## Module template

Create `internal/services/admin/modules/<area>/` with:

- `module.go`: module identity (`ID`) and mount wiring,
- `handlers.go`: area-local handler/service contract consumed by routing,
- `routes.go`: route registration and path ownership for the area prefix,
- `module_test.go`: route contract coverage.

## Authoring rules

- One area owner package per admin area (`modules/<area>`).
- One mounted prefix owner per module under `/app/*`.
- Follow the web-compatible module file shape: `module.go`, `handlers.go`,
  `routes.go`, `module_test.go`.
- Keep area/module tests in the owning `modules/<area>` package. Do not add
  module-scoped tests under `internal/services/admin` root.
- Keep module route declarations explicit with stdlib method+path patterns.
- Use `r.PathValue(...)` for path params instead of manual path splitting.
- Keep route/handler logic transport-thin; move orchestration into area-local
  services when complexity grows.
- Source URLs from `internal/services/admin/routepath` constants/builders.
- Use canonical fragment query (`?fragment=rows`) instead of `/_rows` routes.
- Do not add compatibility aliases for legacy root feature routes.

## Composition rules

- Compose modules through `composition.ComposeAppHandler` and `app.Compose`.
- `app.Compose` validates:
  - module prefix format (`/`, trailing slash),
  - `/app/` prefix ownership,
  - duplicate prefix rejection.

## Removal policy

- Keep only one active route/module path per area.
- Do not reintroduce `internal/services/admin/module/<area>` packages.
- Delete stale route tests and compatibility assertions when behavior is
  intentionally removed.

## Required checks

- `go test ./internal/services/admin/...`
- `make test`
- `make integration`

If templates changed:

- `make templ-generate`

## Definition of done

A module change is done when:

- route ownership is module-local and `/app/*` scoped,
- old aliases and stale paths are removed,
- tests assert current route/method contracts,
- docs remain aligned with the architecture contract.
