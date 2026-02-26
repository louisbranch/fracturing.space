---
name: web-server
description: Web transport boundaries and feature-layer conventions
user-invocable: true
---

# Web Server Conventions

Transport-layer guidance for the Web UI and related services.

## Architecture Rules

- Keep transport handlers thin; orchestration and domain decisions belong in service/domain packages.
- Preserve feature boundaries (`internal/services/web/feature/*`, `internal/services/web2/modules/*`); avoid cross-feature coupling.
- Define interfaces at consumption points and avoid leaking concrete adapters across modules.
- During refactors, prefer clean cutovers over long-lived compatibility routes.

## Routing and Rendering

- Use canonical route path packages for path construction; do not duplicate route constants.
- Keep rendering and template composition scoped to the owning feature.
- Favor explicit request -> service -> response flow over ad hoc handler logic.

## Testing Focus

- Prefer integration tests at transport seams (request mapping, auth checks, response contracts).
- Assert user-visible outcomes and explicit protocol invariants, not incidental markup trivia.

## Architecture Notes

Refer to `docs/project/architecture.md` for system layout and service boundaries.
Refer to `docs/project/domain-language.md` for canonical naming.
