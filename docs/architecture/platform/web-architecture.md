---
title: "Web architecture"
parent: "Platform surfaces"
nav_order: 5
status: canonical
owner: engineering
last_reviewed: "2026-03-09"
---

# Web Architecture

Concise architecture contract for the browser-facing web service.

## Purpose

`web` is a modular BFF that composes route areas while keeping transport wiring
separate from game/auth domain truth. Canonical path: `internal/services/web/`.

## Layering model

- `composition/`: startup composition and module-set assembly.
- `app/`: transport composition primitives.
- `platform/`: cross-cutting HTTP/session/error helpers.
- `module/`: canonical module contract (`Module`, `Mount`, request-scoped resolver types).
- `modules/`: module registry builder, dependency bundles, and area modules (`campaigns`, `dashboard`, `settings`, etc.).
- `routepath/`: canonical route constants, split by owned surface.
- `templates/`: shared layout and templ primitives; area-owned page sets should move out once cross-area ownership becomes unclear. If a feature still uses shared template internals, put a module-owned render seam in front first.

For module internals, areas may be either `transport-only` or `transport + app + gateway` (preferred when orchestration/adapter boundaries are needed).

## Module contract

Each area is one module owner with one mounted prefix. Composition controls which
modules are active and whether they are public or protected.

Required properties:

1. Module boundaries are area-local (no sibling-module coupling).
2. Handlers stay transport-thin; orchestration lives in area-local `app` services.
3. Gateway adapters encapsulate backend protocol mapping.
4. Missing required dependencies fail closed.
5. Composition dependencies are module-owned bundles (`modules.Dependencies.Campaigns`, `Settings`, `PublicAuth`, etc.), not one flat cross-area field bag.
6. Production gateway wiring belongs to the owning area package; the registry may assemble shared cross-cutting inputs and module order, but not feature-local graphs.
7. Optional protected modules should be omitted until fully configured; once selected, module construction must fail fast on missing route-owned services instead of fabricating unavailable placeholders.

## Routing strategy

- Route declarations are module-owned and explicit.
- Canonical browser endpoints come from `routepath` constants/builders.
- Browser URLs owned by `web` are canonically slashless; trailing-slash module prefixes are composition-only subtree mounts and must redirect when a module owns an exact root page.
- `routepath/` stays split by owned surface (`campaigns.go`, `settings.go`, `notifications.go`, etc.) instead of one cross-area route constant file.
- Route-param guards are centralized in reusable helpers (for example
  `withCampaignID`, `withCampaignAndCharacterID`) instead of repeated inline
  path extraction in handlers.
- Form/JSON input parsing is isolated to helper seams; mutation handlers
  orchestrate only request flow + app service calls.
- Protected mutations require authenticated session context.
- Public-auth flows are isolated under public module ownership.
- Username-aware typeahead may be shared across modules, but ownership stays at the service seam:
  - signup availability checks call auth-owned advisory validation endpoints,
  - authenticated invite/mention search calls social-owned ranked people search.
- Legacy top-level invites scaffolding (`/app/invites`) remains intentionally
  unregistered until that area has a production route owner.

## Transport input contracts

- Public JSON handlers must decode with explicit safety guards:
  - bounded request size,
  - unknown-field rejection,
  - single-payload enforcement (reject trailing JSON tokens).
- Form handlers should map request values through dedicated parser helpers and
  keep validation messaging explicit and localized.

## Authorization and mutation boundaries

- Campaign mutation routes require evaluated authorization decisions before
  mutation gateway calls.
- Batch authorization should be used for per-row action visibility.
- Detail/edit/control pages should use true entity reads when the area owns
  them instead of loading a full collection and rediscovering one row in
  transport or render code.
- Transport layers must not approximate permissions from UI fallback logic.
- Chat/game UI routes must consume game-owned communication context for stream
  visibility, persona selection, and scene/session awareness; browser code must
  not derive those rules from transcript bodies.
- Campaign AI automation controls should remain a dedicated campaign automation capability seam; do not couple AI binding UI to participant edit pages just because the GM seat may be AI-controlled.
- Campaign detail pages should render through the area-owned
  `internal/services/web/modules/campaigns/render` seam, not new page-specific `templates` models.
- Browser controls must treat persona selection as message presentation state;
  participant-scoped controls such as gate responses still come from
  authoritative game workflow state.
- The canonical campaign game route (`/app/campaigns/{campaign_id}/game`) is a
  server-rendered game surface that bootstraps `CampaignGameSurface` metadata
  from the game communication service and uses chat websocket delivery only for
  transcript and realtime state updates.

## Principal identity seam

- User-id normalization is centralized in `internal/services/web/platform/userid` and reused by principal/session/viewer and dashboard/webctx seams.
- Shared viewer/language request plumbing is centralized in `internal/services/web/platform/requestresolver`
  so handler bases, page rendering, error rendering, and direct public-page localization follow one request-scoped contract, including localized page-state resolution.
- Root server/composition/module assembly also passes one grouped `requestresolver.PrincipalResolver` contract instead of duplicating flat callback bags at each layer.
- Public modules (`publicauth`, `profile`, `invite`) now consume that same
  grouped principal contract instead of separate signed-in and user-id callbacks.
- Require-vs-optional semantics stay explicit: `userid.Require` for authenticated required boundaries and `userid.Normalize` for optional propagation boundaries.
- Viewer resolver construction is nil-safe to keep package test harnesses deterministic and panic-free.

## Degraded operation model

Fail closed when authz/session/dependency checks are unavailable. Do not keep
placeholder mutation routes or static fake domain data in runtime composition.

## Startup dependency policy

- Startup-blocking integrations are explicit and limited to:
  - `auth`: principal resolution plus auth-owned public/profile/settings flows
  - `social`: principal/profile/settings social metadata plus authenticated
    people-search for invite UX
  - `game`: campaigns and dashboard-sync mutation freshness
- Optional integrations must degrade only the surfaces they own:
  - `ai`: settings AI surfaces and campaign AI affordances
  - `discovery`: public discovery and future public people-browsing surface
  - `userhub`: dashboard and dashboard-sync freshness
  - `notifications`: principal unread badge and notifications module
  - `status`: dashboard service-health surface and reporter flush target
- This policy is owned by `internal/cmd/web/dependency_graph.go`; when startup
  requirements change, update both the policy table and this architecture note.
- Runtime dependency assembly starts in `internal/cmd/web/runtime_dependencies.go`,
  but `internal/services/web/dependencies.go` only coordinates bundle
  construction while `internal/services/web/principal` and owner-local module
  packages bind their own clients; only shared `DashboardSync` freshness
  clients remain centralized there. Keep command-layer code focused on policy,
  addresses, and connection lifecycle, and do not mutate partially-built
  bundles later in `Run`.

## Verification contract

Minimum checks when changing web architecture, modules, or routes:

- `make test`
- `make smoke`
- `make check`

`make check` automatically runs `make web-architecture-check` when web paths changed. Use focused package tests when debugging a specific web slice.
## Deep references

- [Web contributor map](web-contributor-map.md)
- [Web module playbook](../../guides/web-module-playbook.md)
- [Campaign authorization model](campaign-authorization-model.md)
- [Campaign authorization audit and telemetry](../../reference/campaign-authorization-audit.md)
