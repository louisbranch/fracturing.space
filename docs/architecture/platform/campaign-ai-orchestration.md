---
title: "Campaign AI Orchestration"
parent: "Platform surfaces"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-03"
---

# Campaign AI Orchestration

## Purpose

Define the canonical boundary contract for AI game-master control across user
credentials, campaign state authority, and chat relay behavior.

## Responsibilities

- User + AI service:
  - Users own provider credentials (BYO key/OAuth grant).
  - AI service stores provider credentials and references them by opaque
    `ai_agent_id`.
- Game service:
  - Source of truth for campaign-to-agent binding.
  - Source of truth for active campaign session.
  - Source of truth for `ai_auth_epoch` revocation state.
  - Issues signed campaign AI session grants.
- Chat service:
  - Filters participant traffic into AI relay flow.
  - Filters AI responses back to campaign participants.
  - Maintains room AI relay context and refreshes grants.
- AI service:
  - Executes provider calls and orchestration (subagents, MCP augmentation,
    usage accounting).
  - Validates campaign AI session grants before processing turns.

## Internal API Boundary

Game exposes internal-only AI authorization methods via
`game.v1.CampaignAIService`:

- `IssueCampaignAISessionGrant`
- `GetCampaignAIBindingUsage`
- `GetCampaignAIAuthState`

This surface is not a public end-user API. Calls are restricted by
`x-fracturing-space-service-id` and the game-side allowlist
(`FRACTURING_SPACE_GAME_INTERNAL_SERVICE_ALLOWLIST`).

## Session Grant Model

Session grants are short-lived signed tokens with claims:

- `campaign_id`
- `session_id`
- `ai_agent_id`
- `auth_epoch`
- `issued_for_user_id` (optional)
- `jti` / issue + expiry timestamps

The game service issues one grant per room AI context refresh. Chat reuses that
grant for turn submit/subscribe calls until expiry or invalidation. AI does not
need to re-authorize through game every turn when the grant remains valid.

## Revocation and Invalidation

`ai_auth_epoch` is the campaign-level revocation cursor. Grants are valid only
when `claim.auth_epoch == current campaign ai_auth_epoch`.

Epoch rotation triggers:

- `campaign.ai_bind`
- `campaign.ai_unbind`

Session boundaries do not rotate the epoch. Grant validation includes `session_id`
in its claims check, so grants issued for a previous session are automatically
invalid when a new session starts — no epoch bump required.

Epoch does not rotate for unrelated campaign mutations (name/theme/cover/etc.).

## Turn Flow

1. Owner binds an `ai_agent_id` to a campaign.
2. Game persists binding and advances auth epoch.
3. Chat joins/syncs room AI context from game and requests a session grant.
4. Chat forwards participant turn requests to AI with `session_grant`.
5. AI validates token signature/claims and checks auth state (`agent/session/epoch`).
6. AI executes provider turn and streams results.
7. Chat forwards AI outputs to campaign participants.

## Failure Behavior

- Missing/invalid/expired grant: AI rejects turn with permission error.
- Stale epoch/session/agent mismatch: AI rejects as stale precondition.
- Missing binding or inactive session: game refuses grant issuance; chat keeps
  relay disabled for that room.

## Session Start Readiness in AI Modes

For `gm_mode` `ai` and `hybrid`, campaign session start readiness must include:

- bound `ai_agent_id` on campaign metadata
- at least one active participant seat with role `GM` and controller `AI`

If either invariant is missing, campaign readiness reports include stable
session-readiness blocker codes and session start must remain blocked.

## Extensibility

- Built-in first-party AI can use the same `ai_agent_id` + grant contract.
- Provider-specific orchestration (prompt policy, subagents, MCP tools, audit)
  remains AI-service owned and does not leak into game/chat boundaries.
