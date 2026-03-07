---
title: "Campaign session readiness"
parent: "Reference"
nav_order: 12
status: canonical
owner: engineering
last_reviewed: "2026-03-03"
---

# Campaign Session Readiness

Canonical readiness contract used to decide whether a campaign can start a new
session.

## Source of truth

- Domain evaluator: `internal/services/game/domain/readiness/session_start.go`
- Readiness report RPC: `game.v1.CampaignService.GetCampaignSessionReadiness`
- Session start mutation guard: `internal/services/game/api/grpc/game/session_application.go`

## Invariants

A campaign is ready for session start only when all invariants below pass.

Boundary invariants:

- campaign status allows session start (`draft` or `active`)
- no other active session exists

Core invariants:

- at least one active GM participant exists
- at least one active player participant exists
- every active player controls at least one active character
- every active character has a controller participant
- every active character passes game-system readiness checks (when configured)

AI-mode invariants (`gm_mode` `ai` or `hybrid`):

- campaign has a bound `ai_agent_id`
- at least one active participant has role `GM` and controller `AI`

## Blocker codes

- `SESSION_READINESS_CAMPAIGN_STATUS_DISALLOWS_START`
  metadata: `status`
- `SESSION_READINESS_ACTIVE_SESSION_EXISTS`
- `SESSION_READINESS_AI_AGENT_REQUIRED`
- `SESSION_READINESS_AI_GM_PARTICIPANT_REQUIRED`
- `SESSION_READINESS_GM_REQUIRED`
- `SESSION_READINESS_PLAYER_REQUIRED`
- `SESSION_READINESS_PLAYER_CHARACTER_REQUIRED`
  metadata: `participant_name`, `participant_id`
- `SESSION_READINESS_CHARACTER_CONTROLLER_REQUIRED`
  metadata: `character_id`
- `SESSION_READINESS_CHARACTER_SYSTEM_REQUIRED`
  metadata: `character_id`, optional `reason`

## Transport contract

`GetCampaignSessionReadiness` returns:

- `readiness.ready`: boolean
- `readiness.blockers[]`: ordered blockers with:
  - `code` (stable machine-readable code)
  - `message` (localized user-facing text)
  - `metadata` (structured context for clients)

Blocker ordering is deterministic:

1. boundary blockers (`campaign_status_disallows_start`, `active_session_exists`)
2. core/system blockers in canonical evaluation order

## Web behavior

The campaigns sessions page consumes readiness report data and must:

- disable start-session action while readiness is blocked
- display blocker messages so participants can resolve missing requirements
