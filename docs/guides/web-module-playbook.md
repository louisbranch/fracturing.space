---
title: "Web module playbook"
parent: "Guides"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-09"
---

# Web Module Playbook

This playbook defines the default way to add or modify a web area module.

Canonical implementation path: `internal/services/web/`.

## Module Template

Create a package under `internal/services/web/modules/<area>/` with this
baseline:

- `module.go`: module identity and mount implementation.
- `handlers.go`: shared HTTP transport wiring for the area.
- `routes.go`: route registration within the local mux.
- `routes_test.go`: route contract and method coverage.

Supported module archetypes:

- `transport-only`: root package owns route/handler rendering flow without
  dedicated app/gateway subpackages. Use this only while the area remains small
  and has no meaningful orchestration policy.
- `transport + app + gateway`: root package is transport-thin while
  orchestration and transport-adapter mapping live in subpackages.

When a module needs explicit orchestration/adapter boundaries (campaigns/
settings/notifications/dashboard/profile/publicauth are references), split
inside the same area boundary:

- `<area>/`: transport/module surface only (mount, handlers, routes, view maps).
- `<area>/app/`: domain contracts + orchestration logic.
- `<area>/gateway/`: transport adapter integrations (for example gRPC mapping).

When a module has multiple contributor-owned sub-areas, keep that split visible
in the root transport package too:

- use shared files like `handlers.go` and `routes.go` only for package wiring or
  true cross-area helpers,
- move area-owned handler bodies into files such as
  `handlers_profile.go`, `handlers_locale.go`, `handlers_ai_keys.go`,
  `handlers_ai_agents.go`,
- mirror the same ownership in route registration (for example
  `routes_account.go`, `routes_ai.go`) so route edits stay local to the
  transport surface they expose.

For layered modules, carry the same ownership split below transport when the
app/gateway seam stops being cohesive:

- split area-local service methods by owned surface (for example
  `app/service_account.go` and `app/service_ai.go`) instead of keeping one
  catch-all service file,
- split fail-closed gateway behavior the same way so degraded-mode policy stays
  local to the owned surface,
- split gateway adapters by dependency bundle (for example
  `gateway/grpc_account.go` and `gateway/grpc_ai.go`) rather than mixing
  unrelated backend clients behind one broad implementation file.
- when one gateway still spans many operations, store query-side and
  mutation-side dependencies in explicit bundles with narrow capability
  interfaces instead of one flat “everything client” struct. Keep authz checks
  in their own bundle when the module has fail-closed authorization behavior.

## Authoring Rules

- Keep one module prefix owner per area.
- Keep one root package owner per area. Subpackages may exist for that area,
  but sibling area imports remain forbidden.
- Accept service integrations through constructors/interfaces.
- Use one constructor shape per module: `New(Config) Module`.
  Avoid variant constructors (`NewWith...`, option builders, mixed positional
  constructors) because they fragment test and composition seams.
- For campaigns/settings/profile-style modules, build production gateways in
  composition (registry composition files under `modules/registry_*.go`) rather
  than inside `Mount`.
- Runtime module selection is composition-owned: `composition.ComposeAppHandler`
  calls a `modules.RegistryBuilder` with `modules.RegistryInput` to assemble module sets.
  Keep module packages unaware of startup mode flags.
- Modules receive their narrow dependencies at construction time via the
  registry, not through `Mount`.  Protected modules receive a
  `modulehandler.Base` for shared request-scoped resolvers (viewer, user-id,
  language).
- Keep `modules.Dependencies` module-owned and nested by area (for example
  `Dependencies.Campaigns.*`, `Dependencies.Settings.*`). Do not add new
  flat cross-area dependency fields.
- For modules with segmented route ownership, assemble route registration
  through explicit owned slices (for example campaigns:
  overview + participants + characters + character-creation +
  sessions/game + invites) so ownership stays diffable in one place.
- When one module surface depends on multiple backend services with different
  availability profiles, derive health per user-facing surface instead of one
  module-wide backend bit. Hide unavailable sibling links from owned navigation
  and choose `/app/<module>` redirects from the first available surface.
- When a protected module is omitted entirely because its backend is not
  configured, app-shell affordances for that module must also be explicit and
  conditional. Do not keep unconditional nav links to routes that composition
  no longer mounts.
- Keep root module packages transport-thin: handlers/routes own request/response
  flow while orchestration and gateway mapping live in area-local `app` and
  `gateway` subpackages when present.
- When a system-specific workflow includes form parsing or template/view
  mapping, keep workflow registration in the root transport area. `app`
  services may accept a workflow as input for orchestration, but they should
  not also own the workflow registry or transport-facing parser/view methods.
- When those system-specific workflows become a contributor-owned seam of their
  own, move the contract into an area-local subpackage such as
  `<area>/workflow` instead of defining it in the root module package.
- For page-heavy transport areas, prefer explicit per-surface load -> populate
  -> render flow over generic closure/spec scaffolds once contributors need to
  trace behavior route-by-route.
- Keep presentation-specific asset formatting in transport/view seams. `app`
  services may return avatar or media identity fields, but final CDN/static URL
  construction belongs in view mappers or template-facing formatters.
- Keep module-owned browser copy/rendering inside the module area. Do not make
  web modules import sibling service render helpers for user-facing copy; if a
  web surface needs area-specific rendering, add an area-local seam/package
  under `internal/services/web/modules/<area>/`.
- Keep shared templ concerns narrow. `internal/services/web/templates` should
  hold app-shell primitives and small shared helpers; when one area starts
  accumulating a page set that contributors edit together, move that set under
  the owning area instead of growing one cross-area template bucket.
- Temporary root compatibility adapters are migration-only. Remove them in the
  same cutover slice once handlers/modules are wired directly to `app` and
  `gateway` contracts.
- Register routes with stdlib method+path patterns and keep method/path guards
  out of handlers.
- Prefer route-param guard helpers for multi-param routes (for example
  `withCampaignAndCharacterID`) so 404 behavior is centralized and testable.
- Prefer `internal/services/web/platform/routeparam` for single-parameter
  extraction/guard flow instead of repeating trimmed `PathValue` helpers in
  individual modules.
- Prefer route-level contracts that naturally support `HEAD` for `GET`
  surfaces.
- Source browser endpoint URLs from `routepath` constants/builders (including
  script data attributes) instead of hardcoded literals.
- Keep `routepath/` ownership split by area when adding or changing browser
  endpoints. Add route constants/builders to the matching owned file instead of
  reopening a shared monolith.
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
- For form-based mutations, keep form parsing/validation-message creation in
  reusable helper seams instead of repeating inline `ParseForm` branches.
- For public JSON endpoints, decode through strict parser helpers:
  body-size caps, unknown-field rejection, and single-payload enforcement.
- Return typed errors and map them once at transport boundaries.
- Avoid shared global mutable state.
- Protected module defaults must fail closed when a required backend dependency
  is absent; never return placeholder static domain data from runtime module
  wiring.
- For campaign mutation behavior, require evaluated game authorization decisions
  (`AuthorizationService.Can`) before calling mutation gateways.
- For per-row action visibility (for example character editability), use
  `AuthorizationService.BatchCan` with one check per row and map decisions back
  by correlation id.
- Campaign mutation gates must fail closed when authz is unavailable or returns
  an unevaluated decision; do not approximate mutation permissions from
  participant-list fallback logic.
- Do not keep long-lived deferred mutation scaffolds. If a mutation contract is
  not implemented (for example participant update or character control), remove
  transport/app stubs instead of preserving placeholder routes.

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
- Use `internal/services/web/platform/userid.Require` at required-auth app
  boundaries and `internal/services/web/platform/userid.Normalize` for optional
  propagation seams.
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
- Public modules may start with collocated gateway code in `service.go` when
  the gateway surface is small.
  - Once complexity grows, split to explicit `app` + `gateway` packages (for
    example `profile` and `publicauth`).
- Error rendering uses `httpx.WriteJSONError` or custom page helpers instead of
  `weberror.WriteModuleError`, since public routes may serve JSON APIs or
  standalone landing pages rather than app-shell HTML.
- Page rendering uses `pagerender.WritePublicPage` instead of
  `pagerender.WriteModulePage`.

The `publicauth` package is the reference implementation for split-surface
public module ownership plus `app`/`gateway` boundaries.
Public/auth route ownership stays in the root `publicauth` package and is
selected through explicit surface registration owned by composition.

## Registering a Module

1. Implement the module package.
   Ensure the module root has `type Config struct { ... }` and
   `func New(config Config) Module`.
2. Add module constructor wiring in registry composition (`modules/registry_*.go`).
3. Choose public or protected group.
4. Choose route exposure tier:
  - mount only production-ready handlers by default,
  - keep incomplete handlers unregistered until contracts are stable and
    fail-closed checks are in place.
5. Ensure new module dependencies are wired through `modules.RegistryInput` and
   then assigned into the matching `modules.Dependencies.<Area>` bundle.
6. If an area is partially ready, keep one module owner and split route
   registration by explicit surfaces instead of exposing unstable handlers by
   default.
7. Run package tests and architecture checks.

## Required Checks

Run at minimum:

- `go test ./internal/services/web/...`
- `make web-architecture-check`
- `make test`
- `make runtime`
- `make cover`

Coverage must not regress.

## Definition of Done

A module is done when:

- It has an isolated prefix and local mux.
- It has route tests for method and path behavior.
- It does not import sibling modules.
- It is registered in the correct route group.
- Out-of-scope or incomplete routes are not mounted.

## Test Structure Guidance

- Keep `server_test.go` focused on shared server wiring and broad integration
  behavior.
- Split concern-heavy coverage into sibling files such as
  `server_auth_test.go` and `server_static_test.go` to keep review scope tight.
