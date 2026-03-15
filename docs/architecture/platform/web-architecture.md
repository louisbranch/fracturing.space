---
title: "Web architecture"
parent: "Platform surfaces"
nav_order: 5
status: canonical
owner: engineering
last_reviewed: "2026-03-14"
---

# Web Architecture

Concise architecture contract for the browser-facing web service.

## Purpose

`web` is a modular BFF that owns browser routes and page composition while leaving auth, game, social, and other domain truth in backend services. Canonical path: `internal/services/web/`.

## Layering model

- `internal/services/web/`: root ownership seam with package-intent docs and production server assembly.
- `composition/`: startup composition and module-set assembly.
- `app/`: root mux composition and top-level transport policy.
- `platform/`: shared HTTP/session/error/render helpers only.
- `principal/`: canonical request-scoped viewer, locale, and signed-in resolution.
- `module/`: singular module contract only (`Module`, `Mount`, shared `Viewer` shape).
- `modules/`: registry builder, dependency bundles, and feature areas (`campaigns`, `dashboard`, `settings`, etc.).
- `routepath/`: canonical browser paths split by owned surface.
- `templates/`: shared shell/layout primitives only; area-owned page sets should move out once ownership gets blurry.
- Area-local `render/` and `workflow/` packages keep reader-first `doc.go` files plus focused seam tests so contributors can start from handwritten entrypoints instead of generated output.

For module internals, areas may be transport-only or `transport + app + gateway`. Prefer the layered shape once orchestration or backend mapping stops being trivial.

## Module contract

Each area owns one mounted prefix. Composition decides which modules are active and whether they are public or protected.

Required properties:

1. Module boundaries are area-local; sibling modules do not reach into each other.
2. Handlers stay transport-thin; orchestration lives in area-local `app` services.
3. Gateway adapters encapsulate backend protocol mapping.
4. Missing required dependencies fail closed.
5. Composition dependencies are module-owned bundles (`modules.Dependencies.Campaigns`, `Settings`, `PublicAuth`, etc.), not one flat cross-area field bag.
6. Production gateway wiring belongs to the owning area package. The registry may assemble shared runtime inputs and module order, but not feature-local graphs.
7. Shared runtime helpers such as dashboard-sync policy are built once in the registry and passed into area-owned composition entrypoints.
8. Layered module roots depend on ready app services, not raw gateways. `composition.go` builds the production graph so `Mount` stays transport-only.
9. Keep capability and route-surface splits end to end. Do not re-bundle split services at the module root, do not route every request through one catch-all handler receiver, and do not regrow contract sink files after `app` or `gateway` packages are already split.
10. Optional protected modules are omitted until fully configured. Once selected, construction must fail fast on missing route-owned services instead of fabricating unavailable placeholders.

## Routing strategy

- Route declarations are module-owned and explicit.
- Canonical browser endpoints come from `routepath` constants and builders.
- Browser URLs owned by `web` are slashless. Trailing-slash module prefixes are composition-only subtree mounts and must redirect when a module owns an exact root page.
- `routepath/` stays split by owned surface (`campaigns_core.go`, `campaigns_characters.go`, `settings.go`, `notifications.go`, etc.) instead of growing one cross-area path bucket.
- Route-param guards are centralized in reusable helpers such as `withCampaignID` and `withCampaignAndCharacterID`.
- Form and JSON parsing are isolated to helper seams; mutation handlers orchestrate request flow plus app service calls only.
- When one area supports multiple system-specific flows, keep the install-time system manifest in the root area package and derive aliases, defaults, and workflow registration from that registry instead of scattering parser switches across handlers and workflow services.
- Protected mutations require authenticated session context.
- Public auth flows remain isolated under public module ownership.
- Username-aware typeahead may be shared, but service ownership still matters: signup availability checks are auth-owned advisory validation, while authenticated invite or mention search is social-owned people search.
- Legacy top-level invites scaffolding (`/app/invites`) remains intentionally
  unregistered until that area has a production route owner.

## Transport input contracts

- Public JSON handlers decode with explicit safety guards: bounded request size, unknown-field rejection, and single-payload enforcement.
- Form handlers map request values through dedicated parser helpers and keep validation messaging explicit and localized.

## Authorization and mutation boundaries

- Campaign mutation routes require evaluated authorization decisions before mutation gateway calls.
- Batch authorization is preferred for per-row action visibility.
- Detail and control pages use true entity reads when the area owns them instead of loading a full collection and rediscovering one row in transport or render code.
- Transport layers must not approximate permissions from UI fallback logic.
- Chat and game UI routes consume game-owned interaction state for scene awareness, player-phase status, and OOC state; browser code must not derive gameplay authority from transcript bodies.
- Campaign AI automation controls remain a dedicated automation capability seam rather than leaking into participant-edit pages.
- Campaign detail pages render through `internal/services/web/modules/campaigns/render`, not new page-specific `templates` models.
- `/app/campaigns/{campaign_id}/game` is only a `web` launcher: validate access, issue a short-lived `play` launch grant, then hand off browser state, active-play websocket transport, and play-session cookies to `play`.

## Principal identity seam

- User-id normalization is centralized in `internal/services/web/platform/userid` and reused by principal, session, viewer, dashboard, and webctx seams.
- Shared viewer/language request plumbing is centralized in `internal/services/web/principal` so handler bases, page rendering, error rendering, and direct public-page localization all follow one request-scoped contract.
- Root server, composition, and module assembly pass one grouped `principal.PrincipalResolver` contract instead of duplicating flat callback bags at each layer.
- Public modules (`publicauth`, `profile`, `invite`) consume that same grouped principal contract instead of separate signed-in and user-id callbacks.
- Require-vs-optional semantics stay explicit: `userid.Require` for authenticated required boundaries and `userid.Normalize` for optional propagation boundaries.
- Viewer resolver construction is nil-safe so package test harnesses stay deterministic and panic-free.

## Degraded operation model

Fail closed when authz, session, or required dependency checks are unavailable. Do not keep placeholder mutation routes or static fake domain data in runtime composition.

## Startup dependency policy

- Startup policy is service-owned in `internal/services/web/startup_dependencies.go`.
- Command-layer startup code in `internal/cmd/web/dependency_graph.go` consumes that descriptor table, supplies concrete addresses, and fails fast if descriptor coverage drifts.
- Required integrations are limited to `auth`, `social`, and `game`.
- Optional integrations degrade only the surfaces they own: `ai`, `discovery`, `userhub`, `notifications`, and `status`.
- Runtime dependency assembly starts in `internal/cmd/web/runtime_dependencies.go`.
- `internal/services/web/dependencies.go` coordinates bundle construction, while `principal` and owner-local module packages bind their own clients. Shared `DashboardSync` freshness clients are the main remaining centralized cross-area binding.
- Keep command-layer code focused on policy, addresses, and connection lifecycle; do not mutate partially-built bundles later in `Run`.

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
