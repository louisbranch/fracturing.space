---
title: "Web module playbook"
parent: "Architecture"
nav_order: 17
status: canonical
owner: engineering
last_reviewed: "2026-02-28"
---

# Web Module Playbook

This playbook defines the default way to add or modify a web area module.

Path note:

- Current implementation paths are under `internal/services/web/`.
- During package rename, move these packages to `internal/services/web/` while
  preserving the same boundaries.

## Module Template

Create a package under `internal/services/web/modules/<area>/` with this
baseline:

- `module.go`: module identity and mount implementation.
- `handlers.go`: HTTP handlers for the area.
- `service.go`: area orchestration logic.
- `routes.go`: route registration within the local mux.
- `routes_test.go`: route contract and method coverage.

When a module grows beyond one cohesive file set (campaigns is the reference),
split inside the same area boundary:

- `<area>/`: transport/module surface only (mount, handlers, routes, view maps).
- `<area>/app/`: domain contracts + orchestration logic.
- `<area>/gateway/`: transport adapter integrations (for example gRPC mapping).

## Authoring Rules

- Keep one module prefix owner per area.
- Keep one root package owner per area. Subpackages may exist for that area,
  but sibling area imports remain forbidden.
- Accept service integrations through constructors/interfaces.
- For campaigns/settings-style modules, build production gateways in
  composition (`modules/registry.go` via `NewGRPCGateway(...)`) rather than
  inside `Mount`.
- Runtime module selection is composition-owned: `composition.ComposeAppHandler`
  calls `modules.Registry.Build(modules.BuildInput)` to build stable or
  experimental module sets. Keep module packages unaware of startup mode flags.
- When a module intentionally exposes split stable/experimental route sets, expose
  explicit constructors for each surface (`NewStableWithGateway` and
  `NewExperimentalWithGateway`) and avoid ambiguous `NewWithGateway` names.
- Modules receive their narrow dependencies at construction time via the
  registry, not through `Mount`.  Protected modules receive a
  `modulehandler.Base` for shared request-scoped resolvers (viewer, user-id,
  language).
- Keep root module packages transport-thin: handlers/routes own request/response
  flow while orchestration and gateway mapping live in area-local `app` and
  `gateway` subpackages when present.
- Register routes with stdlib method+path patterns and keep method/path guards
  out of handlers.
- Prefer route-level contracts that naturally support `HEAD` for `GET`
  surfaces.
- Source browser endpoint URLs from `routepath` constants/builders (including
  script data attributes) instead of hardcoded literals.
- Emit server-owned app-shell route metadata in layout options so client behavior
  (for example campaign-workspace main styling) is driven by layout contracts,
  not client-side route regexes.
- Use `internal/services/web/platform/httpx.WriteRedirect` for mutation
  success redirects so HTMX and non-HTMX clients stay in parity.
  - Exception: handlers that intentionally render different HTMX vs non-HTMX
    payloads may keep explicit branching; document those cases explicitly with
    tests.
- Use `internal/services/web/platform/httpx.MethodNotAllowed` for `405` +
  `Allow` behavior instead of duplicating module-local helpers.
- Keep handlers thin; call service methods for behavior.
- Return typed errors and map them once at transport boundaries.
- Avoid shared global mutable state.
- Protected module defaults must fail closed when a required backend dependency
  is absent; never return placeholder static domain data from runtime module
  wiring.
- Incomplete/new surfaces must start as experimental module registrations.
  Promotion to default registries is allowed when the exposed route surface is
  stable; unfinished routes must remain unregistered (or explicitly
  experimental).
- For campaign mutation behavior, require evaluated game authorization decisions
  (`AuthorizationService.Can`) before calling mutation gateways.
- For per-row action visibility (for example character editability), use
  `AuthorizationService.BatchCan` with one check per row and map decisions back
  by correlation id.
- Campaign mutation gates must fail closed when authz is unavailable or returns
  an unevaluated decision; do not approximate mutation permissions from
  participant-list fallback logic.

## Security Defaults

- Public auth pages must treat users as authenticated only after validating
  `web_session` through auth service lookup.
- Protected route auth must be session-backed (`web_session`) and validated
  through auth service lookup; do not trust raw user-id headers as an
  authentication source.
- Protected mutation routes (`POST`, `PUT`, `PATCH`, `DELETE`) rely on
  composition-level same-origin checks when requests are cookie-authenticated.
- State-changing auth actions (for example logout) must use non-GET methods.
- Reuse shared request/session helpers (`platform/requestmeta`,
  `platform/sessioncookie`) instead of duplicating cookie/scheme/origin parsing
  logic.

## Request Context Defaults

- Resolve principal/session once per request and reuse that resolved state
  throughout handler flow.
- Use `internal/services/web/platform/webctx.WithResolvedUserID` for
  downstream service calls that require user identity metadata.
- Do not pass raw request context to mutation service calls when resolved user
  identity is available.
- Keep user-scoped service/gateway boundaries explicit: pass `userID`
  parameters instead of extracting identity from transport metadata inside
  gateways.
- Prefer `internal/services/web/platform/weberror.WriteModuleError` for
  consistent localized error rendering across full-page and HTMX app flows.
- Use `internal/services/web/platform/weberror.PublicMessage` for user-visible
  JSON/text errors so raw internal strings are never exposed.

## Public Module Variant

Public (unauthenticated) modules follow a lighter pattern than protected modules:

- They do **not** embed `modulehandler.Base` — there is no authenticated user to
  resolve, so viewer/user-id/language resolvers are not injected.
- Gateway code may be collocated in `service.go` instead of a separate
  `gateway_grpc.go` when the gateway surface is small.
- Error rendering uses `httpx.WriteJSONError` or custom page helpers instead of
  `weberror.WriteModuleError`, since public routes may serve JSON APIs or
  standalone landing pages rather than app-shell HTML.
- Page rendering uses `pagerender.WritePublicPage` instead of
  `pagerender.WriteModulePage`.

The `publicauth` package is the reference implementation for this variant.
Public/auth route ownership is split into explicit surface modules under:
- `internal/services/web/modules/publicauth/surfaces/shell`
- `internal/services/web/modules/publicauth/surfaces/passkeys`
- `internal/services/web/modules/publicauth/surfaces/authredirect`

## Registering a Module

1. Implement the module package.
2. Add module constructor in `internal/services/web/modules/registry.go`.
3. Choose public or protected group.
4. Choose stability tier:
  - experimental (`ExperimentalPublicModules` /
     `ExperimentalProtectedModules`) while scaffolded or incomplete,
  - stable defaults (`DefaultPublicModules` / `DefaultProtectedModules`) once
     exposed routes are production-ready and fail-closed checks are in place.
5. Ensure new module dependencies are wired through `modules.BuildInput` and
   represented in `BuildOutput.Health` when the module implements
   `module.HealthReporter`.
6. If an area is partially ready, keep one module owner and split route
   registration by surface (stable subset vs experimental/additional routes)
   instead of exposing unstable handlers by default.
7. Run package tests and architecture checks.

## Required Checks

Run at minimum:

- `go test ./internal/services/web/...`
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
- It is placed in the correct stability registry (experimental vs stable) for
  the currently exposed route surface.
- Any out-of-scope routes are not mounted in stable mode.

## Test Structure Guidance

- Keep `server_test.go` focused on shared server wiring and broad integration
  behavior.
- Split concern-heavy coverage into sibling files such as
  `server_auth_test.go` and `server_static_test.go` to keep review scope tight.
