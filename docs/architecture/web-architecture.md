---
title: "Web architecture"
parent: "Architecture"
nav_order: 16
status: canonical
owner: engineering
last_reviewed: "2026-03-01"
---

# Web Architecture

## Purpose

`web` is the browser-facing service for user web flows. It is a modular monolith
BFF that composes many route areas without coupling area logic together.

Primary goals:

- Keep transport logic isolated from domain truth.
- Keep each area modular with small muxes.
- Make adding a new area predictable for new contributors.

Current implementation note:

- The canonical web service currently lives under `internal/services/web/`.
- During package rename, move this structure to `internal/services/web/` without
  changing boundaries.

## Layering

Current package layout is organized into four layers:

- `internal/services/web/composition/`: module-set assembly contract and app
  composition input wiring.
- `internal/services/web/app/`: transport composition primitives
  (mounting/auth wrappers/prefix validation).
- `internal/services/web/platform/`: middleware and cross-cutting helpers.
- `internal/services/web/modules/`: area modules (`campaigns`, `dashboard`,
  `discovery`, `notifications`, `profile`, `settings`).
- `internal/services/web/modules/campaigns/app/`: campaigns domain contracts
  and orchestration service logic.
- `internal/services/web/modules/campaigns/gateway/`: campaigns gRPC adapter
  mapping and client integrations.
- `internal/services/web/modules/settings/app/`: settings domain contracts and
  orchestration service logic.
- `internal/services/web/modules/settings/gateway/`: settings gRPC adapter
  mapping and client integrations.
- `internal/services/web/modules/notifications/app/`: notifications domain
  contracts and orchestration service logic.
- `internal/services/web/modules/notifications/gateway/`: notifications gRPC
  adapter mapping and client integrations.
- `internal/services/web/modules/dashboard/app/`: dashboard domain contracts
  and orchestration service logic.
- `internal/services/web/modules/dashboard/gateway/`: dashboard gRPC adapter
  mapping and client integrations.
- `internal/services/web/modules/profile/app/`: profile domain contracts and
  orchestration service logic.
- `internal/services/web/modules/profile/gateway/`: profile gRPC adapter
  mapping and client integrations.
- `internal/services/web/modules/publicauth/`: shared unauthenticated auth flow
  transport/module surface wiring.
- `internal/services/web/modules/publicauth/app/`: public-auth domain
  contracts and orchestration service logic.
- `internal/services/web/modules/publicauth/gateway/`: public-auth gRPC adapter
  mapping and client integrations.
- `internal/services/web/modules/publicauth/surfaces/*`: explicit public/auth
  route-owner modules (`shell`, `passkeys`, `authredirect`).
- `internal/services/web/routepath/`: canonical route constants.

Module archetypes:

- `transport-only`: root package owns route/handler rendering flow directly
  (for example `discovery`).
- `transport + app + gateway`: root package is transport-thin while
  orchestration and adapter mapping live in area-local `app` and `gateway`
  packages.

## Module Model

Each area implements the `module.Module` contract:

- `ID() string`: stable module identity.
- `Mount() (module.Mount, error)`: returns exactly one root prefix and handler.

Composition mounts modules in two groups:

- Public modules (no auth guard).
- Protected modules (auth guard required).

Module registration also has stability tiers:

- Runtime composition uses `modules.Registry.Build(modules.BuildInput)` to
  produce both public/protected module sets and derived service health metadata
  from a single contract. `BuildInput.EnableExperimentalModules` controls
  whether stable-only or stable+experimental module sets are composed.
- The registry decomposes `modules.Dependencies` (gRPC clients) and
  `modules.ModuleResolvers` (request-scoped resolver functions derived from the
  principal resolver) into per-module constructor arguments so individual
  modules receive only the narrow dependencies they need. Each client field is
  typed as the narrow interface defined by the consuming module, so modules
  physically cannot access clients they were not given.
- Runtime startup wiring derives both principal and module inputs from a single
  `web.DependencyBundle`.
- Incomplete/scaffold surfaces stay opt-in through registry build mode rather
  than direct constructor calls.
- Stable modules may intentionally expose only a subset of area routes. Unstable
  routes stay unregistered (or remain in an explicit experimental surface)
  until behavior is production-ready.
- Runtime opt-in for experimental surfaces is explicit through
  `Config.EnableExperimentalModules`.

Startup fails if two modules claim the same prefix.

## Campaign Surface Migration

Campaigns moved to a split-route ownership model to isolate risk:

- Stable surface includes workspace/read/create flows where behavior and navigation
  are considered production-safe.
- Experimental surface hosts incomplete or high-churn campaign routes until they
  pass reliability and permission model stability requirements.
- The split lets the app shell keep stable links and redirects deterministic while
  continuing product development on isolated routes.

## Routing Strategy

Web uses only stdlib routing (`net/http`, `http.ServeMux`).

- Root mux delegates by prefix.
- Each module owns an internal small mux.
- Protected prefixes are wrapped by auth middleware at composition time.

This keeps route ownership explicit and avoids framework lock-in.

## Boundary Rules

- Modules must not import sibling modules. Module-local subpackages are allowed
  when they stay inside one area boundary (for example
  `modules/campaigns/app` and `modules/campaigns/gateway`).
- Cross-cutting code belongs in `platform/*`.
- Path constants belong in `routepath` and nowhere else.
- Module composition prefixes must be canonical (`/` prefix and `/` suffix), and non-canonical prefixes are rejected at compose time.
- Route registration should use stdlib method+path patterns; avoid duplicating
  path/method guards inside handlers.
- `GET` route surfaces should preserve `HEAD` behavior through method+path
  registration.
- Public discovery routes render through shared public page helpers rather than raw
  transport writes.
- Browser-facing script endpoints should be sourced from `routepath` via
  server-rendered data attributes, not hardcoded literals.
- App-shell runtime behavior should consume server-rendered app layout metadata
  (for example campaign-workspace main style policy) rather than inferring route
  ownership from client-side path checks.
- Mutation success redirects should use `platform/httpx.WriteRedirect` so HTMX
  clients receive `HX-Redirect` and browser clients receive `302/Location`.
- Route-level method rejections should use `platform/httpx.MethodNotAllowed` so
  `405` responses keep consistent `Allow` headers.
- App composition may wire modules, but not contain feature logic.
- `composition.ComposeAppHandler` is the runtime boundary that converts
  principal resolvers + module dependencies into `app.ComposeInput`; avoid
  duplicating this wiring in `server.NewHandler`.
- Campaign/settings/notifications/dashboard/profile service gateways are
  composition-owned wiring; modules receive pre-built gateways through
  constructors, not raw client bags.
- Campaigns root package (`modules/campaigns`) is transport-only
  (module mount/routes/handlers/view mapping). Domain orchestration lives in
  `modules/campaigns/app`; gRPC mapping lives in `modules/campaigns/gateway`.
- Settings/notifications/dashboard root packages are transport-only and do not
  keep compatibility shim files after cutover. Domain orchestration and
  fail-closed defaults live in each module `app` package; gRPC mapping lives in
  each module `gateway` package.
- Profile root package is transport-only and does not keep compatibility shim
  files after cutover. Domain orchestration lives in `modules/profile/app`;
  gRPC mapping lives in `modules/profile/gateway`.
- User-scoped gateways/services should accept explicit `userID` parameters;
  avoid hidden transport-metadata extraction inside gateway internals.
- Session cookie and same-origin request proofs are shared platform primitives
  (`platform/sessioncookie`, `platform/requestmeta`) reused by auth and app
  flows.
- Scheme resolution for `requestmeta` is explicit: `X-Forwarded-Proto` is only
  honored when the composed policy sets `requestmeta.SchemePolicy{
  TrustForwardedProto: true }`. In `internal/cmd/web`, this is gated by
  `FRACTURING_SPACE_WEB_TRUST_FORWARDED_PROTO` / `-trust-forwarded-proto`; the
  default is safe for untrusted direct requests.
- User-facing transport errors must resolve to safe public text
  (`platform/weberror.PublicMessage`), never raw backend/internal strings.
- Default app chrome should not link to experimental module routes;
  experimental surfaces are explicitly opt-in.
- Stable campaigns route exposure currently includes
  list/create/overview/participants/characters plus character create,
  character detail, and character-creation apply/reset workflow routes.
- Campaign session/invite mutations (`POST` start/end/create/revoke) are
  mounted only in experimental campaigns modules; stable defaults do not expose
  these mutation handlers.
- Promotion criteria for these routes are tracked in
  `docs/architecture/campaigns-mutation-promotion-checklist.md`.
- Deferred campaign mutation scaffolds (participant update, character
  update/control) are intentionally removed until backed by production-ready
  contracts.
- Scaffold detail surfaces for sessions/invites/game chat remain unregistered
  on stable defaults.
- Campaign cover rendering must resolve to a guaranteed static fallback asset
  (`/static/campaign-cover-fallback.svg`) when upstream assets are absent; this
  avoids runtime references to non-existent static files.
- Campaign mutation flows must enforce authorization through evaluated
  `AuthorizationService.Can` decisions and fail closed when authz decisions are
  missing, unevaluated, or unavailable.
- Campaign list/detail pages should use `AuthorizationService.BatchCan` for
  per-entity action visibility (for example character edit badges) instead of
  issuing N unary auth checks.

## Degraded Operation Strategy

When a gRPC backend dependency is nil at startup, modules degrade according to
their interaction model:

- **Read-only aggregation modules** (dashboard) degrade silently: the
  `unavailableGateway` returns zero-value domain structs so the page renders
  with empty data instead of an error.
- **Modules with user actions** (campaigns, settings, public auth) return
  `apperrors.KindUnavailable` errors.  The user sees a localized error page
  explaining the service is temporarily unavailable.
- **Principal resolution** (viewer, user-id, language) degrades gracefully: nil
  clients fall through to default values (empty user-id, "Adventurer" display
  name, browser-negotiated language).

This distinction keeps the app shell navigable when optional backends are down
while clearly surfacing errors for features that would silently lose data.

## Verification

Architecture guardrails live in tests:

- Prefix uniqueness checks.
- Sibling module import checks.
- Auth wrapping checks for protected module mounts.
- Composition-owned gateway wiring checks for campaigns/settings mount methods.
- Stable vs experimental registry behavior checks.
- Registry build contract checks (stable vs experimental module sets and health
  metadata).
- Shared redirect/method helper checks for HTMX and non-HTMX behavior parity.
- Routepath constant contract checks for module registration patterns.
- Explicit user-id boundary checks for settings gateway/service contracts.
