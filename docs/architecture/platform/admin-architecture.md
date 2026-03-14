---
title: "Admin architecture"
parent: "Platform surfaces"
nav_order: 6
status: canonical
owner: engineering
last_reviewed: "2026-03-07"
---

# Admin Architecture

Concise architecture contract for the operator-facing admin service.

## Purpose

`admin` is the operator web surface for inspection, support, and controlled
operations against game/auth platform services.

Canonical path: `internal/services/admin/`.

## Layering model

- `composition/`: startup composition and module-set assembly.
- `app/`: transport composition primitives.
- `modules/`: area modules (`dashboard`, `campaigns`, `systems`, `catalog`,
  `icons`, `users`, `scenarios`).
- `routepath/`: canonical admin route constants/builders.
- `templates/`: server-rendered UI contracts and shared components.

Legacy area routes under `internal/services/admin/module/<area>` are removed.
Only `internal/services/admin/module` remains as the shared module contract.

## Module contract

Each area owns:

- one module package under `modules/<area>/`,
- one mounted prefix under `/app/*`,
- web-compatible module files (`module.go`, `handlers.go`, `routes.go`,
  `module_test.go`), with area-local route registration and transport dispatch.

Composition owns module set selection and prefix ownership validation.

## Routing strategy

- Canonical authenticated endpoints are `/app/*`.
- Root and legacy top-level feature routes (`/campaigns`, `/systems`, etc.) are
  intentionally unregistered.
- Row/table fragment responses use canonical resource routes with
  `?fragment=rows`; legacy `/_rows` paths are removed.
- Route literals in templates and handlers should resolve from `routepath`
  constants/builders.

## Authorization boundary

- Admin authentication middleware applies only to `/app/*`.
- Static assets (`/static/*`) and non-app paths bypass auth middleware and fall
  through to normal routing behavior.

## Startup dependency visibility

- Admin startup logs deterministic dependency status lines for game, auth, and
  status integrations.
- Capability registration is fail-closed:
  - `admin.game.integration` and `admin.auth.integration` are reported as
    `unavailable` when the initial dependency connection is not established.
- Runtime reconnect loops may recover integrations later; startup capability
  state reflects observed connectivity at initialization time.

## Verification contract

Minimum checks when changing admin architecture, modules, or routes:

- `make test`
- `make smoke`
- `make check`

## Deep references

- [Admin module playbook](../../guides/admin-module-playbook.md)
- [Web architecture](web-architecture.md)
