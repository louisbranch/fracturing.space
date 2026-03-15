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
credentials, game-owned interaction authority, and AI execution.

## Responsibilities

- User + AI service:
  - Users own provider credentials (BYO key/OAuth grant).
  - AI service stores provider credentials and references them by opaque
    `ai_agent_id`.
- Game service:
  - Source of truth for campaign-to-agent binding.
  - Source of truth for active campaign session.
  - Source of truth for active-play scene/player-phase/OOC state.
  - Source of truth for `ai_auth_epoch` revocation state.
  - Issues signed campaign AI session grants.
- Chat service:
  - Provides optional human-only session transcript transport.
  - Validates campaign/session membership for websocket rooms.
  - Is not part of AI pacing, turn submission, or AI response relay.
- AI service:
  - Executes provider calls and orchestration (subagents, MCP augmentation,
    usage accounting).
  - Validates campaign AI session grants before processing turns.
  - Uses MCP tools/resources to inspect and mutate authoritative game state
    once GM automation is enabled for a campaign flow.

## Internal API Boundary

Game exposes internal-only AI authorization methods via
`game.v1.CampaignAIService`:

- `IssueCampaignAISessionGrant`
- `GetCampaignAIBindingUsage`
- `GetCampaignAIAuthState`

This surface is not a public end-user API. Calls are restricted by
`x-fracturing-space-service-id` and the game-side allowlist
(`FRACTURING_SPACE_GAME_INTERNAL_SERVICE_ALLOWLIST`).

Game active-play pacing is exposed through `game.v1.InteractionService`. Chat
may remain as a separate human transcript surface, but it does not participate
in AI turn authority.

## Session Grant Model

Session grants are short-lived signed tokens with claims:

- `campaign_id`
- `session_id`
- `ai_agent_id`
- `auth_epoch`
- `issued_for_user_id` (optional)
- `jti` / issue + expiry timestamps

The game service issues grants for AI execution flows owned by game and AI.
Chat does not request or cache grants.

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
3. Game interaction state reaches an AI-owned decision point, typically when a
   player phase closes and authority returns to the GM.
4. Game or an orchestration worker assembles AI input from authoritative
   interaction state and requests a session grant when needed.
5. AI validates token signature/claims and checks auth state (`agent/session/epoch`).
6. AI executes provider work, using MCP resources for reads and MCP tools for
   authoritative game-state mutations as needed.
7. Game/web surfaces present the resulting GM output through the active-play
   interaction model after those interaction-owned writes are committed.

Current pacing note:

- Human chat transcript is explicitly non-authoritative for AI pacing.
- AI inputs come from committed interaction state, not buffered websocket chat.
- AI control should converge on MCP-driven game mutations rather than direct
  transcript-to-state writes.

## Failure Behavior

- Missing/invalid/expired grant: AI rejects turn with permission error.
- Stale epoch/session/agent mismatch: AI rejects as stale precondition.
- Missing binding or inactive session: game refuses grant issuance and no AI
  turn runs for that interaction state.

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
- Any future AI debug or thought-stream surface is AI-service owned, dev/operator
  only, and separate from the human session transcript.
