---
title: "Web2 module playbook"
parent: "Project"
nav_order: 9
---

# Web2 Module Playbook

This playbook defines the default way to add or modify a web2 area module.

## Module Template

Create a package under `internal/services/web2/modules/<area>/` with this baseline:

- `module.go`: module identity and mount implementation.
- `handlers.go`: HTTP handlers for the area.
- `service.go`: area orchestration logic.
- `routes.go`: route registration within the local mux.
- `routes_test.go`: route contract and method coverage.

## Authoring Rules

- Keep one module prefix owner per area.
- Accept service integrations through constructors/interfaces.
- For campaigns/settings-style modules, build production gateways in composition (`modules/registry.go` via `NewGRPCGateway(...)`) rather than inside `Mount`.
- Consume shared runtime seams from `module.Dependencies` only for per-request/runtime concerns (for example rendering, viewer, user-id, language).
- Register routes with stdlib method+path patterns and keep method/path guards out of handlers.
- Prefer route-level contracts that naturally support `HEAD` for `GET` surfaces.
- Source browser endpoint URLs from `routepath` constants/builders (including script data attributes) instead of hardcoded literals.
- Use `internal/services/web2/platform/httpx.WriteRedirect` for mutation success redirects so HTMX and non-HTMX clients stay in parity.
- Use `internal/services/web2/platform/httpx.MethodNotAllowed` for `405` + `Allow` behavior instead of duplicating module-local helpers.
- Keep handlers thin; call service methods for behavior.
- Return typed errors and map them once at transport boundaries.
- Avoid shared global mutable state.
- Protected module defaults must fail closed when a required backend dependency is absent; never return placeholder static domain data from runtime module wiring.
- Incomplete/new surfaces must start as experimental module registrations. Promotion to default registries is allowed when the exposed route surface is stable; unfinished routes must remain unregistered (or explicitly experimental).
- For campaign mutation behavior, enforce role/access policy in service logic before calling mutation gateways; baseline policy is owner/manager allowed and member denied.

## Security Defaults

- Public auth pages must treat users as authenticated only after validating `web2_session` through auth service lookup.
- Protected route auth must be session-backed (`web2_session`) and validated through auth service lookup; do not trust raw user-id headers as an authentication source.
- Protected mutation routes (`POST`, `PUT`, `PATCH`, `DELETE`) rely on composition-level same-origin checks when requests are cookie-authenticated.
- State-changing auth actions (for example logout) must use non-GET methods.
- Reuse shared request/session helpers (`platform/requestmeta`, `platform/sessioncookie`) instead of duplicating cookie/scheme/origin parsing logic.

## Request Context Defaults

- Resolve principal/session once per request and reuse that resolved state throughout handler flow.
- Use `internal/services/web2/platform/webctx.WithResolvedUserID` for downstream service calls that require user identity metadata.
- Do not pass raw request context to mutation service calls when resolved user identity is available.
- Keep user-scoped service/gateway boundaries explicit: pass `userID` parameters instead of extracting identity from transport metadata inside gateways.
- Prefer `internal/services/web2/platform/weberror.WriteModuleError` for consistent localized error rendering across full-page and HTMX app flows.
- Use `internal/services/web2/platform/weberror.PublicMessage` for user-visible JSON/text errors so raw internal strings are never exposed.

## Registering a Module

1. Implement the module package.
2. Add module constructor in `internal/services/web2/modules/registry.go`.
3. Choose public or protected group.
4. Choose stability tier:
   - experimental (`ExperimentalPublicModules` / `ExperimentalProtectedModules`) while scaffolded or incomplete,
   - stable defaults (`DefaultPublicModules` / `DefaultProtectedModules`) once exposed routes are production-ready and fail-closed checks are in place.
5. If an area is partially ready, keep one module owner and split route registration by surface (stable subset vs experimental/additional routes) instead of exposing unstable handlers by default.
6. Run package tests and architecture checks.

## Required Checks

Run at minimum:

- `go test ./internal/services/web2/...`
- `make test`
- `make integration`
- `make cover`

Coverage must not regress.

## Definition of Done

A module is done when:

- It has an isolated prefix and local mux.
- It has route tests for method and path behavior.
- It does not import sibling modules.
- It is registered in the correct route group.
- It is placed in the correct stability registry (experimental vs stable) for the currently exposed route surface.
- Any out-of-scope routes are not mounted in stable mode.

## Test Structure Guidance

- Keep `server_test.go` focused on shared server wiring and broad integration behavior.
- Split concern-heavy coverage into sibling files such as `server_auth_test.go` and `server_static_test.go` to keep review scope tight.
