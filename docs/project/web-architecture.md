---
title: "Web architecture"
parent: "Project"
nav_order: 8
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

- `internal/services/web/app/`: startup and composition root.
- `internal/services/web/platform/`: middleware and cross-cutting helpers.
- `internal/services/web/modules/`: area modules (`public`, `campaigns`,
  `settings`, etc.).
- `internal/services/web/routepath/`: canonical route constants.

## Module Model

Each area implements the `module.Module` contract:

- `ID() string`: stable module identity.
- `Mount(module.Dependencies) (module.Mount, error)`: returns exactly one root
  prefix and handler.

Composition mounts modules in two groups:

- Public modules (no auth guard).
- Protected modules (auth guard required).

Module registration also has stability tiers:

- Stable defaults are mounted through `modules.DefaultPublicModules()` and
  `modules.DefaultProtectedModules(...)`.
- Incomplete/scaffold surfaces stay opt-in through
  `modules.ExperimentalPublicModules()` and
  `modules.ExperimentalProtectedModules()`.
- Stable modules may intentionally expose only a subset of area routes. Unstable
  routes stay unregistered (or remain in an explicit experimental surface)
  until behavior is production-ready.
- Runtime opt-in for experimental surfaces is explicit through
  `Config.EnableExperimentalModules`.

Startup fails if two modules claim the same prefix.

## Routing Strategy

Web uses only stdlib routing (`net/http`, `http.ServeMux`).

- Root mux delegates by prefix.
- Each module owns an internal small mux.
- Protected prefixes are wrapped by auth middleware at composition time.

This keeps route ownership explicit and avoids framework lock-in.

## Boundary Rules

- Modules must not import sibling modules.
- Cross-cutting code belongs in `platform/*`.
- Path constants belong in `routepath` and nowhere else.
- Route registration should use stdlib method+path patterns; avoid duplicating
  path/method guards inside handlers.
- `GET` route surfaces should preserve `HEAD` behavior through method+path
  registration.
- Browser-facing script endpoints should be sourced from `routepath` via
  server-rendered data attributes, not hardcoded literals.
- Mutation success redirects should use `platform/httpx.WriteRedirect` so HTMX
  clients receive `HX-Redirect` and browser clients receive `302/Location`.
- Route-level method rejections should use `platform/httpx.MethodNotAllowed` so
  `405` responses keep consistent `Allow` headers.
- App composition may wire modules, but not contain feature logic.
- Campaign/settings service gateways are composition-owned wiring; `Mount` must
  not construct gateways from `module.Dependencies` clients.
- User-scoped gateways/services should accept explicit `userID` parameters;
  avoid hidden transport-metadata extraction inside gateway internals.
- Session cookie and same-origin request proofs are shared platform primitives
  (`platform/sessioncookie`, `platform/requestmeta`) reused by auth and app
  flows.
- User-facing transport errors must resolve to safe public text
  (`platform/weberror.PublicMessage`), never raw backend/internal strings.
- Default app chrome should not link to experimental module routes;
  experimental surfaces are explicitly opt-in.
- Campaigns are stable-by-default for read/create workspace flows; in-campaign
  mutation routes remain hidden until participant permission policy is
  finalized.
- Stable campaigns route exposure currently includes
  list/create/overview/participants/characters; scaffold detail surfaces
  (sessions, invites, game chat, character detail) remain unregistered.
- Campaign mutation flows must enforce authorization through evaluated
  `AuthorizationService.Can` decisions and fail closed when authz decisions are
  missing, unevaluated, or unavailable.
- Campaign list/detail pages should use `AuthorizationService.BatchCan` for
  per-entity action visibility (for example character edit badges) instead of
  issuing N unary auth checks.

## Verification

Architecture guardrails live in tests:

- Prefix uniqueness checks.
- Sibling module import checks.
- Auth wrapping checks for protected module mounts.
- Composition-owned gateway wiring checks for campaigns/settings mount methods.
- Stable vs experimental registry behavior checks.
- Shared redirect/method helper checks for HTMX and non-HTMX behavior parity.
- Routepath constant contract checks for module registration patterns.
- Explicit user-id boundary checks for settings gateway/service contracts.
