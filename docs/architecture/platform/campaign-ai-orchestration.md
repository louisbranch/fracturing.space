---
title: "Campaign AI Orchestration"
parent: "Platform surfaces"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-13"
---

# Campaign AI Orchestration

## Purpose

Define the canonical boundary contract for AI game-master control across user
credentials, game-owned interaction authority, and AI execution.

## Responsibilities

- Users own provider credentials; AI stores and dereferences the backing
  runtime templates.
- Game is the source of truth for campaign binding, active session,
  interaction state, AI GM participant authority, and `ai_auth_epoch`; it
  issues signed campaign AI session grants.
- Chat may carry human transcript traffic, but it is not part of AI pacing,
  turn submission, or AI response relay.
- AI validates grants, resolves the live backing runtime, and executes the
  provider + MCP orchestration loop.

## Internal API Boundary

Game exposes internal-only AI authorization via
`game.v1.CampaignAIService` (`IssueCampaignAISessionGrant`,
`GetCampaignAIBindingUsage`, `GetCampaignAIAuthState`). Calls are restricted by
`x-fracturing-space-service-id` and
`FRACTURING_SPACE_GAME_INTERNAL_SERVICE_ALLOWLIST`.

Game active-play pacing remains on `game.v1.InteractionService`.

AI exposes `ai.v1.CampaignOrchestrationService.RunCampaignTurn` as the
campaign-only execution path. It is intentionally separate from the user-facing
`ai.v1.InvocationService.InvokeAgent` RPC so campaign turns can enforce grant
validation, MCP augmentation, and provider tool policy.

## Session Grant Model

Session grants are short-lived signed tokens carrying campaign/session scope,
the current `participant_id`, `auth_epoch`, optional `issued_for_user_id`, and
normal JWT identity + lifetime fields.

The grant proves campaign/session/participant scope plus the current revocation
cursor. It does not embed the reusable runtime template identity; AI resolves
that from live `GetCampaignAIAuthState` data after validation.

## Revocation and Invalidation

`ai_auth_epoch` is the campaign-level revocation cursor. Grants are valid only
when `claim.auth_epoch == current campaign ai_auth_epoch`.

Epoch rotation happens on `campaign.ai_bind` and `campaign.ai_unbind`.
Session changes do not rotate the epoch; session-scoped claims make older
grants stale automatically. Unrelated campaign edits do not rotate the epoch.

Credential/provider-grant revocation is blocked while a `DRAFT` or `ACTIVE`
campaign still depends on that auth reference. Agent bind eligibility requires
an `active` agent plus a ready credential/provider-grant reference.

## Turn Flow

1. Owner binds a backing AI runtime template to a campaign.
2. Game persists binding and advances auth epoch.
3. Game interaction reaches an AI-owned decision point, including
   `session.started` bootstrap turns before an active scene exists.
4. Game/worker requests a session grant for the active campaign session.
5. AI validates the grant, checks live auth state, sets MCP context, rebuilds a
   fresh session brief, and executes provider work against the curated tool set.
6. The model must commit authoritative GM narration through interaction-owned
   MCP tools before the turn succeeds.
7. Worker completes the AI turn only after those authoritative writes land;
   otherwise it records failure in game-owned lifecycle state.

## Initial AI MCP Policy

The MVP orchestration path is OpenAI-first and MCP-backed. `set_context` stays
orchestrator-owned, the prompt brief is rebuilt from authoritative MCP
resources on every turn, and the model receives only the GM-safe subset of MCP
tools needed for scene bootstrap, interaction pacing, GM narration commit, and
Daggerheart rules/dice support. Campaign lifecycle, participant CRUD, fork, and
other non-GM mutations are intentionally excluded.

Exact prompt-brief inputs and bootstrap behavior live in
[Campaign AI Session Bootstrap](campaign-ai-session-bootstrap.md). Human chat
transcript remains non-authoritative for pacing; AI control reads and writes
through committed interaction state.

## Failure Behavior

- Missing/invalid/expired grant: AI rejects turn with permission error.
- Stale epoch/session/participant mismatch: AI rejects as stale precondition.
- Missing binding or inactive session: game refuses grant issuance and no AI
  turn runs for that interaction state.
- Provider/MCP/orchestration errors after the turn starts: worker records the
  turn as failed through `FailAIGMTurn`; retries then flow through explicit
  game-owned retry behavior rather than duplicate outbox handling.

## Session Start Readiness in AI Modes

For `gm_mode` `ai` and `hybrid`, campaign session start readiness must include:

- bound `ai_agent_id` on campaign metadata
- at least one active participant seat with role `GM` and controller `AI`

If either invariant is missing, campaign readiness reports include stable
session-readiness blocker codes and session start must remain blocked.

## Extensibility

- Built-in first-party AI can use the same campaign-scoped grant contract.
- Provider-specific orchestration (prompt policy, subagents, MCP tools, audit)
  remains AI-service owned and does not leak into game/chat boundaries.
- Any future AI debug/thought-stream surface stays AI-service owned,
  dev/operator only, and separate from the human transcript.
- Richer bootstrap material such as summaries, notes, imported files, and
  operator approvals should remain orchestrator-owned additions on top of the
  same campaign/session/participant grant boundary.
