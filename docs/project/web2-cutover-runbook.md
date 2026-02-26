---
title: "Web2 Cutover Runbook"
parent: "Project"
nav_order: 14
---

# Web2 Cutover Runbook

## Purpose

Define the minimum safe procedure for switching primary user traffic from `web` to `web2`, validating behavior during the switch, and rolling back quickly if regressions appear.

## Preconditions

- `make test`, `make integration`, and `make cover` are green on the candidate commit.
- Cutover smoke matrix test is green:
  - `go test ./internal/services/web2 -run TestCutoverSmokeMatrixUnauthenticatedRoutes`
- `web2` is configured with stable defaults (`EnableExperimentalModules=false`) for production cutover.
- On-call owner for auth + web2 is assigned for the cutover window.

## Smoke Checklist

Run this checklist immediately before switch, after partial traffic, and after full traffic.

Automated baseline:

- `go test ./internal/services/web2 -run TestCutoverSmokeMatrixUnauthenticatedRoutes`
  - Compares legacy `web` and `web2` unauthenticated route behavior for campaigns/settings/invites/notifications.
  - Keeps known parity gaps explicit instead of silently drifting.
- `go test ./internal/services/web -run TestCutoverSmokeMatrixAuthenticatedJourneyParity`
  - Compares legacy `web` and `web2` authenticated journey outcomes for campaign navigation plus current invites/notifications parity gaps.

Manual journey checks:

1. Auth/session: login, logout, and session revalidation from a fresh browser profile.
2. Campaigns: campaigns list, campaign overview, sessions, participants, characters, invites pages.
3. Settings/profile: profile + locale pages render and save flows return expected status.
4. Invites and notifications: verify current expected behavior for the selected surface (legacy parity or explicit gap).
5. Public pages: discovery and public profile routes render from `web2` stable defaults.

## Traffic Switch Procedure

1. Confirm preconditions and announce cutover start in ops channel.
2. Deploy the target `web2` commit and verify health endpoint responses.
3. Shift ingress to `web2` in stages (example: 10% -> 50% -> 100%).
4. At each stage, run the smoke checklist and confirm no auth/session or campaign regressions.
5. If all checks pass at 100%, declare cutover complete and keep elevated monitoring for one release window.

## Rollback Procedure

1. Immediately route traffic back to `web`.
2. Re-run smoke checks against `web` to confirm baseline behavior restored.
3. Disable further `web2` rollouts until issue triage is complete.
4. Capture incident notes: first bad stage, failing journey, request IDs, and rollback timestamp.
5. Open follow-up issue(s) and link evidence before the next cutover attempt.

## Evidence to Store

- Command output for `make test`, `make integration`, `make cover`.
- Smoke checklist results with timestamps and operator initials.
- Rollback evidence (if triggered): ingress change record, failure symptom, mitigation confirmation.
