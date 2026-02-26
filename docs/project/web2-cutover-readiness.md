---
title: "Web2 Cutover Readiness"
parent: "Project"
nav_order: 13
---

# Web2 Cutover Readiness

## Purpose

Track the remaining gates before switching primary user-facing traffic from `web` to `web2`.

`web` remains the production user-facing service until every gate in this document is satisfied or explicitly waived.

Current module policy note:

- Stable-by-default `web2` routing mounts only stable modules.
- Incomplete/scaffold surfaces are opt-in via experimental module registration until parity is complete.

## Open Blockers

1. **Feature data parity for app modules**
   - Replace remaining unavailable/scaffold app adapters (invites/notifications/profile) with real service adapters.
   - Preserve fail-closed defaults while wiring adapters (missing dependencies must surface `503`, never placeholder success).
   - Ensure auth metadata propagation and user scoping match `web` behavior.
   - Promote completed app modules from experimental registration into stable default registries.

2. **Campaign workspace parity**
   - Keep default campaigns surface stable (implemented read/create flows) and keep unresolved in-campaign mutation routes unexposed.
   - Baseline mutation permission policy is now encoded (owner/manager allowed, member denied); expand only with explicit policy sign-off.
   - When mutation routes are reintroduced, keep authorization and redirects equivalent to `web` semantics.

3. **App-wide localization parity**
   - Extend EN/PT-BR keys and rendering beyond auth shell into app module content.

## Recently Closed

- **Public surface parity baseline**
  - Discovery/public profile are mounted in stable default public modules.
- **Cutover confidence baseline**
  - Added automated smoke matrix comparison tests:
    - `go test ./internal/services/web2 -run TestCutoverSmokeMatrixUnauthenticatedRoutes`
    - `go test ./internal/services/web -run TestCutoverSmokeMatrixAuthenticatedJourneyParity`
  - Added cutover + rollback operations guide:
    - `docs/project/web2-cutover-runbook.md`

## Recommended Execution Order

1. Service-backed adapters for invites/notifications/profile/settings.
2. Campaign workspace detail-page parity (read paths first, then mutations).
3. App-wide i18n parity pass.
4. Dual-stack smoke expansion (authenticated journeys) + rollback drill using `docs/project/web2-cutover-runbook.md`.

## Acceptance Gate for Switching Traffic

- All blockers in this document are resolved or explicitly waived with risk sign-off.
- `make test`, `make integration`, and `make cover` are green on parity changes.
- End-to-end smoke checks pass for:
  - login/logout/session revalidation,
  - campaign navigation and intentionally exposed campaign actions,
  - invites and notifications flows,
  - settings/profile and locale switching.
- Rollback drill is documented and validated before traffic switch.

## Evidence Pointers

- Web2 composition/root wiring: `internal/services/web2/server.go`
- Web2 module boundaries: `internal/services/web2/modules/`
- Cutover smoke matrix test: `internal/services/web2/cutover_smoke_matrix_test.go`
- Authenticated dual-stack smoke matrix test: `internal/services/web/cutover_smoke_matrix_authenticated_test.go`
- Cutover and rollback runbook: `docs/project/web2-cutover-runbook.md`
- App shell template: `internal/services/web2/templates/layout.templ`
- Auth shell templates: `internal/services/web2/templates/public_auth.templ`
- Route constants: `internal/services/web2/routepath/routepath.go`
