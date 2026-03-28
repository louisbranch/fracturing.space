---
title: "AI service lifecycle terms"
parent: "Reference"
nav_order: 21
status: canonical
owner: engineering
last_reviewed: "2026-03-23"
---

# AI service lifecycle terms

Stable vocabulary for contributors working in the AI service.

## Auth reference

An **auth reference** is the typed selector stored on an AI agent that answers
"which backing auth source does this agent use at runtime?"

- Owning package: `internal/services/ai/agent`
- Code shape: `agent.AuthReference`
- Valid kinds:
  - `credential`
  - `provider_grant`

Why it exists:

- Agent workflows now carry one typed auth-reference value through transport,
  service, and storage instead of translating through split credential/grant
  IDs.
- The `agent` domain owns the exclusivity rule: an agent must use exactly one
  auth source when required.

Contributor rule:

- Change auth-reference validation, normalization, or vocabulary in
  `internal/services/ai/agent`.
- Transport and storage should project the typed value, not reimplement the
  exclusivity rules.

## Credential

A **credential** is a user-owned BYO provider secret that AI stores in sealed
form and opens only at call time.

- Owning package: `internal/services/ai/credential`
- Typical use: direct API-key style provider auth

Contributor rule:

- Plaintext secret validation belongs in the `credential` domain.
- Encryption and decryption belong at the service/storage boundary, not in the
  domain package.

## Provider grant

A **provider grant** is a user-owned OAuth-backed provider authorization that
can be refreshed and revoked independently of agent definitions.

- Owning package: `internal/services/ai/providergrant`
- Typical use: OpenAI OAuth-based runtime access

### Refresh lifecycle

The provider-grant domain owns the refresh vocabulary and transitions.

Statuses:

- `active`: ready for provider calls
- `refresh_failed`: the latest refresh attempt failed and the grant is not
  currently usable
- `expired`: token lifetime elapsed without a usable refresh path
- `revoked`: owner explicitly disabled the grant

Transitions:

- `RecordRefreshSuccess(...)` writes the new token material, clears
  `LastRefreshError`, updates `RefreshedAt`, and returns the grant to
  `active`.
- `RecordRefreshFailure(...)` records the refresh failure detail, updates
  `RefreshedAt`, and moves the grant to `refresh_failed`.
- Transport code may decide when to attempt a refresh, but it should persist
  the result by applying the provider-grant domain transition first.

Contributor rule:

- Refresh-state semantics belong in `internal/services/ai/providergrant`.
- Do not add ad-hoc refresh status mutations in transport or sqlite code.

## Access request

An **access request** is an owner-gated approval record for delegated agent
invocation.

- Owning package: `internal/services/ai/accessrequest`
- Typical use: requester asks an agent owner for invoke access

Lifecycle terms:

- `review_*` fields (`reviewer_user_id`, `review_note`, `reviewed_at`) belong to
  the approval or denial decision taken while a request is still pending.
- `revoke_*` fields (`revoker_user_id`, `revoke_note`, `revoked_at`) belong to
  the later removal of previously approved access.

Contributor rule:

- Keep approval/denial history separate from revoke history.
- Do not overload review metadata to represent revocation in service, storage,
  or transport code.

## Typed session brief

A **typed session brief** is the authoritative prompt input collected for one
campaign turn before prompt rendering.

- Owning package: `internal/services/ai/orchestration`
- Code shape: `orchestration.SessionBrief`
- Collected by: `ContextSourceRegistry` through `ContextSource.Collect`
- Rendered by: `PromptRenderer`

Why it exists:

- The prompt path no longer re-parses rendered prompt sections to recover
  bootstrap or interaction state.
- Context collection and prompt rendering are separate seams, which makes
  prompt behavior easier to test and replace.

Contributor rule:

- Add new authoritative prompt inputs through context sources and brief
  collection.
- Do not infer runtime state back out of already rendered prompt text.

## Bootstrap mode

**Bootstrap mode** is the special case where the collected session brief shows
there is no active scene yet for the current interaction state.

- Detection: `SessionBrief.Bootstrap()`
- Behavior: the prompt renderer emits bootstrap authority guidance and the tool
  surface allows opening-scene creation/activation plus GM narration commit

Contributor rule:

- Bootstrap detection belongs to typed interaction-state collection, not to
  string inspection in prompt rendering.

## Prompt render policy

The **prompt render policy** is the explicit runtime configuration that decides
which static instruction text and closing guidance the renderer applies.

- Owning package: `internal/services/ai/orchestration`
- Code shape: `orchestration.PromptRenderPolicy`
- Chosen in: `internal/services/ai/app`

Contributor rule:

- Composition-root defaults belong in `internal/services/ai/app`.
- Renderer behavior should consume the explicit policy it is given instead of
  reaching back into configuration or instruction-loading code.

## Related docs

- [AI service contributor map](ai-service-contributor-map.md)
- [Campaign AI orchestration](../architecture/platform/campaign-ai-orchestration.md)
- [Campaign AI session bootstrap](../architecture/platform/campaign-ai-session-bootstrap.md)
