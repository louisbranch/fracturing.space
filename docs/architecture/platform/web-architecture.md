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
separate from game/auth domain truth.

Canonical path: `internal/services/web/`.

## Layering model

- `composition/`: startup composition and module-set assembly.
- `app/`: transport composition primitives.
- `platform/`: cross-cutting HTTP/session/error helpers.
- `module/`: canonical module contract (`Module`, `Mount`, request-scoped resolver types).
- `modules/`: module registry builder, dependency bundles, and area modules (`campaigns`, `dashboard`, `settings`, etc.).
- `routepath/`: canonical route constants, split by owned surface.
- `templates/`: shared layout and templ primitives; area-owned page sets should
  move out once cross-area ownership becomes unclear.

For module internals, areas may be either:

- `transport-only`, or
- `transport + app + gateway` (preferred when orchestration/adapter boundaries are needed).

## Module contract

Each area is one module owner with one mounted prefix. Composition controls which
modules are active and whether they are public or protected.

Required properties:

1. Module boundaries are area-local (no sibling-module coupling).
2. Handlers stay transport-thin; orchestration lives in area-local `app` services.
3. Gateway adapters encapsulate backend protocol mapping.
4. Missing required dependencies fail closed.
5. Composition dependencies are module-owned bundles (`modules.Dependencies.Campaigns`, `Settings`, `PublicAuth`, etc.), not one flat cross-area field bag.

## Routing strategy

- Route declarations are module-owned and explicit.
- Canonical browser endpoints come from `routepath` constants/builders.
- `routepath/` stays split by owned surface (`campaigns.go`, `settings.go`,
  `notifications.go`, etc.) instead of one cross-area route constant file.
- Route-param guards are centralized in reusable helpers (for example
  `withCampaignID`, `withCampaignAndCharacterID`) instead of repeated inline
  path extraction in handlers.
- Form/JSON input parsing is isolated to helper seams; mutation handlers
  orchestrate only request flow + app service calls.
- Protected mutations require authenticated session context.
- Public-auth flows are isolated under public module ownership.
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
- Transport layers must not approximate permissions from UI fallback logic.
- Chat/game UI routes must consume game-owned communication context for stream
  visibility, persona selection, and scene/session awareness; browser code must
  not derive those rules from transcript bodies.
- Browser controls must treat persona selection as message presentation state;
  participant-scoped controls such as gate responses still come from
  authoritative game workflow state.
- The canonical campaign game route (`/app/campaigns/{campaign_id}/game`) is a
  server-rendered game surface that bootstraps `CampaignGameSurface` metadata
  from the game communication service and uses chat websocket delivery only for
  transcript and realtime state updates.

## Principal identity seam

- User-id normalization is centralized in `internal/services/web/platform/userid`
  and reused by principal/session/viewer and dashboard/webctx seams.
- Require-vs-optional semantics are explicit:
  - `userid.Require` for authenticated required user-id boundaries,
  - `userid.Normalize` for optional request-scoped propagation boundaries.
- Viewer resolver construction is nil-safe for user-id resolver wiring to keep
  package test harnesses deterministic and panic-free.

## Degraded operation model

Fail closed when authz/session/dependency checks are unavailable.

Do not keep placeholder mutation routes or static fake domain data in runtime
composition.

## Startup dependency policy

- Startup-blocking integrations are explicit and limited to:
  - `auth`: principal resolution plus auth-owned public/profile/settings flows
  - `social`: principal/profile/settings social metadata
  - `game`: campaigns and dashboard-sync mutation freshness
- Optional integrations must degrade only the surfaces they own:
  - `ai`: settings AI surfaces and campaign AI affordances
  - `discovery`: discovery public surface
  - `userhub`: dashboard and dashboard-sync freshness
  - `notifications`: principal unread badge and notifications module
- This policy is owned by `internal/cmd/web/dependency_graph.go`; when startup
  requirements change, update both the policy table and this architecture note.
- Runtime dependency assembly is owned by `internal/cmd/web/runtime_dependencies.go`.
  Assemble the full server dependency graph there before calling `web.NewServer`;
  do not bootstrap a bundle and mutate it later in `Run`.

## Verification contract

Minimum checks when changing web architecture, modules, or routes:

- `go test ./internal/services/web/...`
- `make web-architecture-check`
- `make integration`

## Deep references

- [Web contributor map](web-contributor-map.md)
- [Web module playbook](../../guides/web-module-playbook.md)
- [Campaign authorization model](campaign-authorization-model.md)
- [Campaign authorization audit and telemetry](../../reference/campaign-authorization-audit.md)
