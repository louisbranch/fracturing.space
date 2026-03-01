---
title: "Campaigns mutation promotion checklist"
parent: "Architecture"
nav_order: 18
status: canonical
owner: engineering
last_reviewed: "2026-03-01"
---

# Campaigns Mutation Promotion Checklist

This checklist defines the exit criteria for promoting campaigns mutation routes
from experimental to stable surface ownership.

## Scope

Routes covered:

- `POST /app/campaigns/{campaignID}/sessions/start`
- `POST /app/campaigns/{campaignID}/sessions/end`
- `POST /app/campaigns/{campaignID}/invites/create`
- `POST /app/campaigns/{campaignID}/invites/revoke`

## Promotion Criteria

All items must pass before stable registration:

- Authorization behavior is fail-closed and verified:
  - missing authz backend denies access,
  - authz transport errors deny access,
  - unevaluated authz decisions deny access.
- Validation behavior is explicit and localized:
  - malformed form payloads return `400` with stable localization keys,
  - required fields (`session_id`, `participant_id`, `invite_id`) return `400`
    with stable localization keys.
- Stable/experimental route contracts are verified:
  - stable campaigns module keeps these mutation routes unregistered,
  - experimental campaigns module exposes these mutation routes.
- Redirect behavior is verified for both clients:
  - browser requests return `302 Location`,
  - HTMX requests return `HX-Redirect` parity where applicable.
- Gateway dependency behavior is verified:
  - missing session/invite clients return typed unavailable errors,
  - mutation transport failures map to user-safe error keys/messages.

## Required Verification

- `go test ./internal/services/web/modules/campaigns/...`
- `go test ./internal/services/web/...`
- `make test`
- `make integration`
- `make cover`

## Promotion Procedure

1. Move the four mutation route registrations from
   `registerExperimentalRoutesForCampaigns` to stable route registration.
2. Keep contract tests that explicitly assert stable/experimental ownership.
3. Update `docs/architecture/web-architecture.md` and
   `docs/architecture/web-module-playbook.md` to reflect the new stable
   surface.
4. Remove this checklist once promotion is complete and represented in canonical
   architecture docs.
