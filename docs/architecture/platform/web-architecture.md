---
title: "Web architecture"
parent: "Platform surfaces"
nav_order: 5
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
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
- `modules/`: area modules (`campaigns`, `dashboard`, `settings`, etc.).
- `routepath/`: canonical route constants.

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

## Routing strategy

- Route declarations are module-owned and explicit.
- Canonical browser endpoints come from `routepath` constants/builders.
- Protected mutations require authenticated session context.
- Public-auth flows are isolated under public module ownership.
- Legacy top-level invites scaffolding (`/app/invites`) remains intentionally
  unregistered until that area has a production route owner.

## Authorization and mutation boundaries

- Campaign mutation routes require evaluated authorization decisions before
  mutation gateway calls.
- Batch authorization should be used for per-row action visibility.
- Transport layers must not approximate permissions from UI fallback logic.

## Degraded operation model

Fail closed when authz/session/dependency checks are unavailable.

Do not keep placeholder mutation routes or static fake domain data in runtime
composition.

## Verification contract

Minimum checks when changing web architecture, modules, or routes:

- `go test ./internal/services/web/...`
- `make web-architecture-check`
- `make integration`

## Deep references

- [Web module playbook](../../guides/web-module-playbook.md)
- [Campaign authorization model](campaign-authorization-model.md)
- [Campaign authorization audit and telemetry](../../reference/campaign-authorization-audit.md)
