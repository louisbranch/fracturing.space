---
title: "Web module playbook"
parent: "Guides"
nav_order: 4
status: canonical
owner: engineering
last_reviewed: "2026-03-23"
---

# Web Module Playbook

This playbook defines the default way to add or modify a web area module.

Canonical implementation path: `internal/services/web/`.

Use the canonical [Web testing map](../architecture/platform/web-testing-map.md)
when deciding whether a change belongs in root web coverage, module route
coverage, or cross-module architecture guardrails.

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

Default composition guidance:

- Every module defines a `CompositionConfig` struct and a
  `Compose(config CompositionConfig) module.Module` entrypoint in
  `composition.go`. Struct parameters make call sites self-documenting and
  prevent positional-arg transposition.
- Modules that may be unconfigured at startup use
  `ComposeProtected(options, deps) (module.Module, bool)` so the registry
  can skip them.
- Keep optional mounting checks in the central registry. Small modules should
  not replicate `configured()` boilerplate just to restate the same nil checks.
- Reserve heavier composition shapes (multiple surface configs, sub-surface
  service builders) for modules that genuinely need them: multiple route
  surfaces, per-surface availability policy, or route-owned service graphs.

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
- if one module owns multiple separately-mounted public surfaces, expose an
  area-owned surface-set composition entrypoint so the central registry does
  not need to know the module’s internal surface list or ordering.

For layered modules, carry the same ownership split below transport when the
app/gateway seam stops being cohesive:

- split area-local service methods by owned surface (for example
  `app/service_account.go` and `app/service_ai.go`) instead of keeping one
  catch-all service file,
- keep exported app service methods with the owned capability files too. Do
  not route every public method through one `service_exports.go` bucket once
  the package is already split by capability,
- when one transport-owned page workflow assembles multiple reads, keep the
  aggregation in the transport-owned page service and expose explicit app reads
  for each input instead of introducing one bundled page-data contract,
- when one workflow package must normalize or filter those reads before
  rendering, keep any assembled intermediate state local to the workflow
  package rather than exporting an app-owned aggregate type just to feed
  templates,
- if a workflow package still needs app-owned read DTOs, adapt them once at
  the workflow boundary and keep the workflow contract on workflow-owned
  types after that point instead of threading app-owned page data through
  every system-specific workflow implementation,
- keep app-layer constructor config explicit by owned capability too. When one
  area still has one service package, prefer nested capability configs over one
  flat read/mutation dependency bag so each constructor documents what it
  actually consumes,
- once that service package grows large, split contracts, capability configs,
  and concrete service builders across focused files instead of keeping one
  `service_contracts.go` megafile,
- keep those focused config and builder files capability-owned too. After the
  first split, do not replace one `service_contracts.go` sink with one
  `service_config.go` or `service_builders.go` sink that mixes unrelated
  catalog, participant, character, invite, and creation wiring again,
- apply the same rule to production composition. Once one area needs helper
  files under `composition`, split generated-client grouping and app-service
  config assembly by capability instead of keeping one `composition_services.go`
  sink with a second cross-capability wiring bag,
- once one module has explicit capability configs, do not reintroduce broad
  exported "read gateway"/"mutation gateway" bag interfaces or convenience
  constructors on top of them just to shorten tests,
- keep test config helpers aligned with production composition too. If
  production wiring moved from one aggregate gateway constructor to explicit
  capability constructors, update module tests to build the same explicit
  capability config instead of preserving the deleted aggregate shape,
- if package tests still need one aggregate seam for concise fixtures, keep it
  in `_test.go` helpers only. Do not leave combined gateway bundles or
  convenience constructors in production app packages once explicit capability
  configs exist,
- apply the same rule in adapter packages. If a `gateway` package keeps
  explicit capability constructors for production wiring, move any remaining
  aggregate constructor used only by package tests into `_test.go` instead of
  keeping it exported from production files,
- apply the same rule to partially-normalized modules too. If a module still
  exposes a broad gateway at the module root for compatibility with older
  wiring, keep any `newService(gateway)`-style test shortcut in `_test.go`
  rather than leaving it in production app files,
- once transport depends on split capability services (for example account vs
  AI), remove any leftover exported aggregate app `Service` or `ServiceConfig`
  surface and update callers/tests to build the owned capability services
  directly,
- apply the same rule to route-surface groupings too. If transport already
  depends on separate page/session/passkey/recovery services, do not keep a
  broad exported app `Service` constructor just to re-bundle them in module
  roots or tests,
- after that split, keep module roots on owned handler-service groups too.
  Composition may still assemble those groups from a gateway, but `Module`
  config and `Mount` should not keep raw gateway fields once the route surface
  already depends on explicit app-service groups,
- if transport already depends directly on owned capability services rather
  than one route-surface bundle, pass those services straight through module
  config and handler wiring. Do not add a second rebundling layer just to
  shorten composition code,
- when an area offers both collection and entity reads, detail/edit/control
  transport should use true entity reads instead of loading the full collection
  and rediscovering one row in transport or render code,
- when app-layer constructors expose one transport-facing capability service,
  keep the constructor signature scoped to the exact capability config and any
  explicit cross-cutting seams it consumes (for example unary authz) instead
  of passing the whole module `ServiceConfig` back into every constructor,
- once a capability config declares sibling reads it depends on (for example
  participant-owned workspace policy reads or character-owned participant
  roster reads), use those owned inputs inside the service implementation
  instead of reaching back through broader package-level helper seams,
- when one “workspace” or “page context” read seam starts carrying
  session/game/workflow-specific data, split that into a dedicated capability
  interface instead of widening the generic area-summary contract,
- if one settings/configuration seam only needs participant reads as a fallback
  for authorization or editor state, keep those reads on the participant seam
  instead of widening the configuration seam with participant ownership,
- if one editor or mutation surface stops fitting the surrounding ownership
  seam, split it into a dedicated capability service instead of splitting the
  read and write halves across unrelated services,
- if one same-noun capability still mixes list/detail reads, ownership-state
  decisions, and destructive writes after that first split, separate read,
  ownership, and mutation services so transport and tests trace one interaction
  mode at a time,
- carry that same split through gateway contracts and dependency bundles. If
  character CRUD and character-ownership routes are separate app services, do not
  keep one shared character-mutation gateway or one shared mutation dep bundle
  underneath them,
- apply the same rule to participant-style governance seams too. If one
  capability still mixes collection/detail editor reads with create/update
  governance writes, split read/editor state from mutation ownership so
  transport traces page loads separately from access-changing writes,
- do the same for session and invite workflows. If one capability still mixes
  detail/list readiness or typeahead reads with lifecycle or create/revoke
  writes, split read surfaces from mutation ownership so GET and POST paths
  stay independently traceable,
- when participant-adjacent UI owns campaign automation controls, keep that as
  a dedicated automation capability seam instead of widening participant or
  campaign-configuration services with automation ownership,
- apply the same split to gateway contracts. Do not keep dead or empty
  sibling read/mutation gateway interfaces around once the owned capability
  has moved to its own seam,
- split fail-closed gateway behavior the same way so degraded-mode policy stays
  local to the owned surface,
- split gateway adapters by dependency bundle (for example
  `gateway/grpc_account.go` and `gateway/grpc_ai.go`) rather than mixing
  unrelated backend clients behind one broad implementation file.
- when one gateway still spans many operations, store query-side and
  mutation-side dependencies in explicit bundles with narrow capability
  interfaces instead of one flat “everything client” struct. Keep authz checks
  in their own bundle when the module has fail-closed authorization behavior.
- if one gateway package still needs one aggregate dependency entrypoint for
  startup or tests, keep that aggregate made of capability-owned dep structs
  (for example `CatalogReadDeps`, `InviteReadDeps`, `AuthorizationDeps`) and
  do not reintroduce flat `Read`/`Mutation` mega-bags.

## Authoring Rules

- Keep one module prefix owner per area.
- Keep one root package owner per area. Subpackages may exist for that area,
  but sibling area imports remain forbidden.
- Accept service integrations through constructors/interfaces.
- Use one constructor shape per module: `New(Config) Module`.
  Avoid variant constructors (`NewWith...`, option builders, mixed positional
  constructors) because they fragment test and composition seams.
- For production modules, keep startup wiring in an area-owned
  `composition.go` entrypoint that the registry calls. The registry should
  select modules and pass dependency policy, not build feature-local gateways
  inline or move gateway construction into `Mount`.
- For small one-surface modules, let that entrypoint take exact positional
  dependencies instead of repeating an area-local `CompositionConfig` wrapper
  around one or two clients and shared helpers.
- For layered modules, keep `Module` config on ready transport-facing app
  services plus shared handler dependencies. `composition.go` should build the
  gateway and service graph; `Mount` should stay focused on local mux and
  handler wiring.
- When one layered area still needs many backend clients at startup, group the
  composition-owned gateway inputs by the same owned route surfaces that the
  module exports instead of one flat `CompositionConfig` client bag.
- If multiple areas share one runtime helper policy, build it once in the
  registry (for example dashboard-sync freshness) and pass the ready helper to
  module composition. Do not reconstruct the same helper inside each area.
- Optional module selection belongs in the registry. Small modules may rely on
  explicit nil checks there instead of repeating module-local `ComposePublic`
  or `ComposeProtected` wrappers when the constructor only forwards exact deps.
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
  overview + participants + characters +
  sessions/game + invites) so ownership stays diffable in one place.
  When those slices need different transport helpers, bind owned handler
  values per slice instead of routing all methods through one root handler
  receiver.
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
- For JSON endpoints, use `platform/httpx.DecodeJSONStrictInvalidInput`
  instead of module-local malformed-body wrappers so size limits,
  unknown-field rejection, trailing-token handling, and stable
  invalid-input mapping stay shared.
- When a system-specific workflow includes form parsing or template/view
  mapping, keep workflow registration in the root transport area. `app`
  services may accept a workflow as input for orchestration, but they should
  not also own the workflow registry or transport-facing parser/view methods.
- Keep that workflow registration install-time and manifest-driven. The same
  root transport registry should own user-facing aliases/defaults for form
  parsing so adding one system does not require another handler-local or
  workflow-local parser switch.
- When those system-specific workflows become a contributor-owned seam of their
  own, move the contract into an area-local subpackage such as
  `<area>/workflow` instead of defining it in the root module package.
- Once an area owns `render/` or `workflow/` subpackages, keep those packages
  reader-first: add a package-intent `doc.go` and focused seam tests on the
  exported entrypoints so contributors can start there instead of generated
  `*_templ.go` output.
- If that workflow subpackage starts handling both page assembly and mutation
  orchestration, split those into separate services/interfaces so GET and POST
  transport paths depend on the narrower workflow surface they actually use.
- For page-heavy transport areas, prefer explicit per-surface load -> populate
  -> render flow over generic closure/spec scaffolds once contributors need to
  trace behavior route-by-route.
- When one render package starts routing section-specific pages back through a
  shared marker or switch-driven template, split those templates by owned
  surface so changing participants, characters, sessions, or invites stays
  local to one file set.
- Do not reintroduce a broad internal adapter view just to feed those split
  templates. Keep section templates typed to their owned page view or narrow
  render contract, and pass only the specific helper inputs they need.
- Apply the same ownership rule to render helpers. After section templates are
  split, move overview/participants/characters/sessions/invites helpers into
  owned files instead of keeping one cross-section helper bucket.
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
- Keep module coverage local and durable:
  - route contract checks in the owning `routes_test.go`
  - handler/page branching in owned `handlers*_test.go`
  - cross-module boundary expectations in
    `internal/services/web/modules/architecture_test.go` or
    `internal/services/web/modules/boundary_guardrails_test.go`
- When a guardrail only needs to protect constructor/package ownership, prefer
  AST-backed invariants on imports, struct fields, and constructor calls over
  raw source-fragment string checks.
- Do not add placeholder files or filename-specific guardrails just to pin a
  layering story in place. Protect the package contract so contributors can
  refactor file splits without first updating ceremonial scaffolding.
- Prefer route-param guard helpers for multi-param routes (for example
  `withCampaignAndCharacterID`) so 404 behavior is centralized and testable.
- Reuse `internal/services/web/platform/httpx.ReadRouteParam` or
  `WithRequiredRouteParam` for simple single-parameter extraction when it
  already matches the route contract. Do not add another shared helper package
  for one-off `PathValue` wrapping that only one area needs.
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
- For form-based mutations and posted settings forms, use shared
  `platform/httpx` form helpers when multiple areas need the same parsing and
  error-mapping policy. Keep one-off parsing local instead of introducing a new
  shared wrapper package.
- For public JSON endpoints, decode through strict parser helpers:
  body-size caps, unknown-field rejection, and single-payload enforcement.
- Return typed errors and map them once at transport boundaries.
- Avoid shared global mutable state.
- Protected module defaults must fail closed when a required backend dependency
  is absent; never return placeholder static domain data from runtime module
  wiring.
- If composition selects a protected module for mounting, missing required
  route-owned services should fail fast during `New`/`Mount` validation instead
  of being silently backfilled with unavailable placeholders. If a module is
  truly optional, make registry composition omit it until its full dependency
  set is present.
- For campaign mutation behavior, require evaluated game authorization decisions
  (`AuthorizationService.Can`) before calling mutation gateways.
- For per-row action visibility (for example character editability), use
  `AuthorizationService.BatchCan` with one check per row and map decisions back
  by correlation id.
- Keep unary mutation-gate authz and batch row-hydration authz as separate
  constructor dependencies when a module owns both. Do not route list/detail
  batch authorization back through the same broad service config field used for
  mutation guards.
- Campaign mutation gates must fail closed when authz is unavailable or returns
  an unevaluated decision; do not approximate mutation permissions from
  participant-list fallback logic.
- Do not keep long-lived deferred mutation scaffolds. If a mutation contract is
  not implemented (for example participant update or character ownership), remove
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
- Use `internal/services/web/principal.WithResolvedUserID` for
  downstream service calls that require user identity metadata.
- Do not pass raw request context to mutation service calls when resolved user
  identity is available.
- Keep user-scoped service/gateway boundaries explicit: pass `userID`
  parameters instead of extracting identity from transport metadata inside
  gateways.
- At composition/module boundaries, prefer one grouped
  `internal/services/web/principal` principal contract over separate
  `ResolveSignedIn`, `ResolveUserID`, `ResolveLanguage`, or `ResolveViewer`
  callback fields.
- Prefer `internal/services/web/platform/weberror.WriteModuleError` for
  consistent localized error rendering across full-page and HTMX app flows.
- Use `internal/services/web/platform/weberror.PublicMessage` for user-visible
  JSON/text errors so raw internal strings are never exposed.

## Public Module Variant

Public (unauthenticated) modules follow a lighter pattern than protected modules:

- They do **not** embed `modulehandler.Base`.
- Public modules that need shared page/localization/signed-in state should use
  `publichandler.Base` built from `principal.PrincipalResolver`.
- Request-time signed-in detection and optional viewer/user-id/language
  resolution come from that shared principal seam, not ad hoc gateway or
  session-validation checks inside the module.
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

- `make test`
- `make smoke`
- `make check`

Use `make web-architecture-check` and focused web package tests when you need
web-specific diagnostics during iteration. `make check` automatically runs the
web architecture gate when web paths changed. Use `make cover` only when you
need standalone coverage output separate from `make check`.

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
