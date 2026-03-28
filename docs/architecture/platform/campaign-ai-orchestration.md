---
title: "Campaign AI Orchestration"
parent: "Platform surfaces"
nav_order: 3
status: canonical
owner: engineering
last_reviewed: "2026-03-23"
---

# Campaign AI Orchestration

## Purpose

Define the boundary for campaign-scoped AI GM control across game authority,
provider-backed AI execution, and committed interaction writes.

## Responsibilities

- Users own provider credentials and grants; AI dereferences the live runtime.
- Game is the source of truth for campaign binding, active session,
  interaction state, AI GM participant authority, and `ai_auth_epoch`.
- AI validates grants, resolves the bound agent/runtime, and executes the
  provider + tool loop through direct gRPC calls to game.
- Human chat transcript is not authoritative for pacing or AI turn completion.

## API Boundary

Game exposes internal-only AI authorization through `game.v1.CampaignAIService`
(`IssueCampaignAISessionGrant`, `GetCampaignAIBindingUsage`,
`GetCampaignAIAuthState`). Calls are restricted by
`x-fracturing-space-service-id` and
`FRACTURING_SPACE_GAME_INTERNAL_SERVICE_ALLOWLIST`.

AI exposes `ai.v1.CampaignOrchestrationService.RunCampaignTurn` as the campaign
execution path. It stays separate from `ai.v1.InvocationService.InvokeAgent` so
campaign turns can enforce grant validation, tool augmentation, and orchestration
policy.

Both invocation surfaces may carry provider usage accounting and an optional
`reasoning_effort` override when the backing provider supports it.

## Session Grant Contract

Session grants are short-lived signed tokens carrying:

- campaign and session scope
- current `participant_id`
- current `auth_epoch`
- optional `issued_for_user_id`
- normal JWT identity and lifetime fields

The grant proves campaign/session authority and revocation cursor. It does not
embed the reusable runtime template; AI resolves that from live
`GetCampaignAIAuthState` data after grant validation.

## Revocation Rules

- `ai_auth_epoch` is the campaign-level revocation cursor.
- Grants are valid only when `claim.auth_epoch == current ai_auth_epoch`.
- Epoch rotation happens on `campaign.ai_bind` and `campaign.ai_unbind`.
- Session changes invalidate grants through session-scoped claims, not epoch
  rotation.
- Credential or provider-grant revoke is blocked while an active or draft
  campaign still depends on that auth reference.

## Turn Flow

1. Owner binds an AI agent to the campaign.
2. Game persists the binding and advances `ai_auth_epoch`.
3. An AI-owned interaction decision point is reached, including bootstrap turns.
4. Game or worker requests a session grant for the active campaign session.
5. AI validates the grant, checks live auth state, opens a fixed-authority game
   session, rebuilds the current brief, and runs provider + tool orchestration.
6. The model must commit authoritative GM narration through interaction-owned
   tools before the turn succeeds.
7. Worker completes the turn only after those writes land; otherwise it records
   failure in game-owned lifecycle state.

## Tool and Prompt Policy

The runtime is still Daggerheart-first. The production surface is a curated
subset of GM-safe tools for scene bootstrap, interaction pacing, authoritative
GM narration, campaign artifacts, and Daggerheart rules/reference reads.
Campaign lifecycle, participant CRUD, fork, and similar non-GM mutations stay
out of the tool profile.

Turn completion is controlled by an explicit `TurnPolicy` seam in the generic
runner. The context-source registry owns source naming, per-source tracing
spans, and duplicate typed-state rejection so brief assembly is explicit rather
than order-dependent.

The always-on prompt collector comes from
`orchestration.NewCoreContextSourceRegistry()`. The composition root then adds
Daggerheart-specific context sources from `orchestration/daggerheart/`.

Campaign-context ownership is split cleanly:

- `campaigncontext/`
  Artifact defaults and path policy
- `campaigncontext/instructionset/`
  Instruction-file loading and composition
- `campaigncontext/memorydoc/`
  `memory.md` structure helpers
- `campaigncontext/referencecorpus/`
  Read-only system reference search and read
- `orchestration/daggerheart/`
  Current Daggerheart prompt/context sources

## Failure Behavior

- Missing, invalid, or expired grant: permission failure
- Stale epoch/session/participant mismatch: stale precondition failure
- Missing binding or inactive session: game refuses grant issuance
- Provider or orchestration failure after turn start: worker records
  `FailAIGMTurn`; retries remain game-owned behavior

## Session-Start Readiness

For `gm_mode` `ai` or `hybrid`, campaign session start readiness requires:

- a bound `ai_agent_id`
- at least one active `GM` participant seat controlled by `AI`

If either invariant is missing, readiness returns stable blocker codes and
session start remains blocked.

## Extensibility

- Built-in first-party AI can use the same campaign-scoped grant contract.
- Provider-specific orchestration policy stays AI-service owned; it does not
  leak into game or chat boundaries.
- Persisted debug traces are AI-owned. Cross-process live fanout remains future
  work.
- Additional game systems are future architecture work. The current production
  prompt and tool surfaces are intentionally optimized for the Daggerheart-first
  runtime that exists today.
