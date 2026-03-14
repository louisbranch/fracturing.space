---
title: "Campaign AI Orchestration"
parent: "Platform surfaces"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-09"
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
  - Consumes game-owned communication context to know which streams/personas are available to each participant connection.
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

Game also exposes `game.v1.CommunicationService` for transport/UI surfaces that
need caller-specific stream visibility and persona eligibility. AI grant flow
and communication context remain separate contracts: the former authorizes AI
execution, the latter authorizes participant-facing communication routing.

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

Credential/provider-grant revocation is blocked while the revoked auth reference
is still used by any agent bound to a `DRAFT` or `ACTIVE` campaign. This keeps
campaign bindings explicit and prevents silent breakage from a settings-side
revoke.

Agent bind eligibility requires both:

- persisted agent lifecycle state `active`
- a ready auth reference (`credential` or `provider grant`) that is neither
  revoked nor otherwise unavailable

## Turn Flow

1. Owner binds an `ai_agent_id` to a campaign.
2. Game persists binding and advances auth epoch.
3. Chat joins/syncs room AI context from game and requests a session grant.
4. Chat buffers ordinary participant transcript locally; it does not submit every
   chat message to AI.
5. A new GM handoff request is the default pacing trigger. When chat opens a new
   `gm_handoff` control gate in an AI-enabled room, it submits one buffered turn
   payload to AI with `session_grant`.
6. AI validates token signature/claims and checks auth state (`agent/session/epoch`).
7. AI executes provider turn and streams results.
8. Chat forwards AI outputs to campaign participants.

Current pacing note:

- The buffered handoff payload is a chat-owned aggregate of recent participant
  transcript plus optional handoff reason text.
- This is a transitional transport contract. The long-term authority seam is
  still an explicit game-owned control workflow plus a richer AI input contract,
  not per-message relay.

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
