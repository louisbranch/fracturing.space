---
title: "Campaign AI Orchestration"
parent: "Platform surfaces"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-19"
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
  provider + tool orchestration loop via direct gRPC calls to the game service.

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
validation, tool augmentation, and provider tool policy.

Both RPCs now accept an optional `reasoning_effort` override for providers that
support explicit reasoning levels, and both responses can carry provider usage
accounting (`input_tokens`, `output_tokens`, `reasoning_tokens`,
`total_tokens`) when the backing adapter reports it.

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
5. AI validates the grant, checks live auth state, opens a direct gRPC
   session with fixed campaign/session/participant authority, rebuilds a fresh
   session brief, and executes provider work against the curated tool set.
6. The model must commit authoritative GM narration through interaction-owned
   tools before the turn succeeds.
7. Worker completes the AI turn only after those authoritative writes land;
   otherwise it records failure in game-owned lifecycle state.

## Tool Policy

The orchestration path is OpenAI-first with direct gRPC tool dispatch. The
prompt path now collects a typed session brief from authoritative game
resources on every turn before rendering the final provider-facing prompt,
session authority is fixed per turn, and the model receives only the GM-safe
subset of tools needed for scene bootstrap, interaction pacing, GM narration
commit, campaign artifacts, and Daggerheart rules/reference support. Campaign
lifecycle, participant CRUD, fork, and other non-GM mutations are intentionally
excluded from the production tool profile.

The composition root chooses the concrete prompt render policy explicitly. AI
startup loads instruction files, builds the context-source collector, and then
constructs the prompt renderer with the canonical closing/interaction policy.
When instruction files are partially unavailable, only the missing instruction
field degrades to inline defaults; the typed brief collector and configured
context-source registry remain active.

Campaign-context support is split by ownership:

- `campaigncontext` owns artifact defaults and artifact path policy
- `campaigncontext/instructionset` owns instruction-file loading/composition
- `campaigncontext/memorydoc` owns `memory.md` structure helpers
- `campaigncontext/referencecorpus` owns read-only game-system reference search/read

Exact prompt-brief inputs and bootstrap behavior live in
[Campaign AI Session Bootstrap](campaign-ai-session-bootstrap.md). Human chat
transcript remains non-authoritative for pacing; AI control reads and writes
through committed interaction state.

## Failure Behavior

- Missing/invalid/expired grant: AI rejects turn with permission error.
- Stale epoch/session/participant mismatch: AI rejects as stale precondition.
- Missing binding or inactive session: game refuses grant issuance and no AI
  turn runs for that interaction state.
- Provider/orchestration errors after the turn starts: worker records the
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
- Provider-specific orchestration (prompt policy, subagents, tools, audit)
  remains AI-service owned and does not leak into game/chat boundaries.
- Any future AI debug/thought-stream surface stays AI-service owned,
  separate from the human transcript, even when `play` forwards an alpha
  browser view of persisted turn-debug updates over its websocket.
- Richer bootstrap material such as summaries, notes, imported files, and
  operator approvals should remain orchestrator-owned additions on top of the
  same campaign/session/participant grant boundary.
