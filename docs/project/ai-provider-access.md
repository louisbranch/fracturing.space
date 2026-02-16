---
title: "AI Provider Access"
parent: "Project"
nav_order: 14
---

# AI Provider Access and Campaign Assignment

This document defines the service boundaries and domain model for user-managed
AI provider access:

- Bring-your-own (BYO) API credentials.
- Provider OAuth grants for API usage.
- Access-request workflows for shared/team-managed credentials.
- Campaign-level assignment of an AI to run GM workflows.

This is a specification document for implementation planning, not a guarantee of
all listed APIs in the current release.

## Goals

- Let users add multiple AI credentials per provider (OpenAI first).
- Support both direct credential entry and provider OAuth grant flows.
- Keep campaign gameplay state/event authority in game service.
- Keep credential/token lifecycle ownership out of game/auth identity domains.
- Make campaign AI assignment explicit and auditable.

## Non-goals

- Defining full GM behavior or narrative quality policy.
- Storing provider secrets in game or transport services.
- Re-calling provider APIs during replay of historical game events.

## Service Boundaries

Boundary decision:

- **Auth service** (`internal/services/auth/`):
  - Owns user identity, login credentials, and first-party OAuth issuance.
  - May continue to support external login identity linking.
  - Does not own gameplay AI provider credential lifecycle.
- **AI service** (`internal/services/ai/`, planned):
  - Owns provider credential/grant lifecycle, secure storage, refresh, and
    provider adapters (OpenAI first).
  - Owns AI agent/profile configuration built from those credentials.
  - Owns provider-facing invocation interfaces for higher-level services.
- **Game service** (`internal/services/game/`):
  - Owns campaign authority and event-sourced assignment state.
  - Stores references to AI resources (for example `agent_id`), not secrets.
  - Persists accepted AI outputs as campaign events when they affect canon.
- **Transport services** (`web`, `mcp`, `chat`, `admin`):
  - Orchestrate UX and API calls.
  - Do not directly persist provider secrets or mutate game projections.

## Domain Language

- **Provider Credential**: User-managed API access material for a provider. Can
  be a direct API token or a reference to OAuth-backed grant material.
- **Provider Grant**: OAuth authorization result for provider API usage, with
  scope and refresh lifecycle.
- **AI Agent**: Named configuration that binds model/provider defaults to one
  credential/grant owner context.
- **Access Request**: Request from one user to use a credential/agent owned by
  another user or organization context.
- **Campaign AI Assignment**: Event-sourced campaign fact that links campaign GM
  automation mode to an AI agent reference.

## Resource Ownership and Storage

| Resource | Owning service | Storage authority | Notes |
|---|---|---|---|
| User identity | `auth` | `data/auth.db` | Existing identity model remains unchanged. |
| Provider credentials/grants | `ai` | `data/ai.db` (planned) | Encrypted-at-rest secrets, refresh lifecycle, audit metadata. |
| AI agent configs | `ai` | `data/ai.db` (planned) | References credentials/grants by ID. |
| Access requests/approvals | `ai` | `data/ai.db` (planned) | Tracks requester, owner, scope, status, reviewer metadata. |
| Campaign AI assignment refs | `game` | `game-events` + projections | Event-sourced references only (no provider secrets). |

## Core Flows

## 1) Add BYO API credential

1. User submits provider + label + API token via web/admin UX.
2. Transport calls AI service credential endpoint.
3. AI service validates format, optionally performs lightweight provider check,
   encrypts secret, stores metadata, and returns credential ID.

## 2) Connect provider via OAuth grant

1. User starts provider OAuth flow from transport UX.
2. AI service issues provider redirect, handles callback, exchanges code, stores
   grant material (encrypted), and records granted scopes.
3. AI service returns a provider grant ID that can back one or more AI agents.

Note: this OAuth flow is for provider API authorization, separate from auth
service login/oauth issuance concerns.

## 3) Request access to shared owner-managed AI

1. Requester selects owner-managed credential or agent and submits request.
2. AI service stores `PENDING` access request with requested scope/constraints.
3. Owner/admin approves or denies.
4. On approval, AI service records grant of use rights without exposing raw
   secret material to requester.

## 4) Assign AI to campaign GM role

1. Campaign admin chooses an accessible AI agent.
2. Transport calls game service assignment command with `agent_id` reference.
3. Game service validates permission and emits assignment event.
4. Game projections expose current assignment status and reference metadata.

## 5) Runtime AI invocation

1. Game/chat workflows determine an AI action is needed.
2. Caller requests AI service invocation using the campaign-assigned `agent_id`.
3. AI service resolves credential/grant, executes provider request, and returns
   result + usage metadata.
4. If result mutates campaign canon, game service persists derived event(s);
   replay consumes events and does not re-call provider APIs.

## Invariants

1. Provider secrets are only stored and managed by AI service.
2. Game service stores AI assignment references, never provider tokens.
3. Accepted gameplay effects are persisted as game events, not as provider
   response dependencies.
4. Replay must be deterministic from persisted events and projections.
5. Access control is server-enforced; client hints never grant new privileges.
6. Credential use and assignment changes are auditable.

## Security and Compliance Baseline

- Encrypt credential/grant secrets at rest.
- Redact secrets from logs, traces, and error messages.
- Enforce provider- and feature-scoped permissions.
- Track credential provenance and last-used metadata.
- Support rotation/revocation without campaign record corruption.

## Provider OAuth Grant Contract

## ProviderGrant model

Represents one user-owned OAuth authorization to a provider API.

Required fields:

- `id`: stable grant identifier.
- `owner_user_id`: user that owns grant.
- `provider`: provider enum (`OPENAI` first).
- `status`: lifecycle status.
- `granted_scopes`: normalized scope list.
- `token_ciphertext`: encrypted token payload (access token, refresh token,
  token type, expiry metadata).
- `created_at`, `updated_at`.
- `last_refreshed_at` (optional).
- `expires_at` (optional, provider-supplied).
- `revoked_at` (optional).
- `last_refresh_error` (optional, non-secret text only).

Token payload is never returned by read APIs.

## ProviderConnectSession model

Represents one in-progress OAuth handshake.

Required fields:

- `id`: stable connect-session identifier.
- `owner_user_id`.
- `provider`.
- `state_hash`: hashed CSRF state token.
- `code_verifier_ciphertext`: encrypted PKCE verifier.
- `requested_scopes`.
- `status`: `pending` or `completed`.
- `expires_at`, `created_at`, `updated_at`.

Session expiry is enforced by `expires_at` checks; sessions are one-time by
behavioral contract (`pending -> completed`).

## ProviderGrant lifecycle states

`ProviderGrant.status` values:

- `active`: usable for provider API calls.
- `expired`: provider token is expired and refresh did not restore usability.
- `refresh_failed`: refresh attempted and failed.
- `revoked`: explicitly revoked by owner/admin; unusable.

Allowed transitions:

- `active -> refresh_failed`
- `active -> expired`
- `active -> revoked`
- `refresh_failed -> active`
- `refresh_failed -> expired`
- `refresh_failed -> revoked`
- `expired -> active` (only through successful refresh)
- `expired -> revoked`

Disallowed transitions:

- Any `revoked -> *`

## ProviderGrantService contract (`ai.v1`)

## `StartProviderConnect`

Starts OAuth handshake.

Request:

- `provider`
- `requested_scopes[]`

Response:

- `connect_session_id`
- `state`
- `authorization_url`
- `expires_at`

Behavior:

- Validates authenticated caller and provider support.
- Creates one `ProviderConnectSession`.
- Generates state + PKCE material.
- Returns provider authorization URL with required parameters.

## `FinishProviderConnect`

Completes OAuth handshake after callback details are received.

Request:

- `connect_session_id`
- `state`
- `authorization_code`

Response:

- `provider_grant` (metadata only; no token secrets)

Behavior:

- Validates session ownership, status, expiration, and one-time use.
- Verifies `state` and PKCE context.
- Exchanges code with provider token endpoint.
- Encrypts and stores token payload in `ProviderGrant`.
- Marks connect session `completed`.

## `ListProviderGrants`

Lists grants owned by caller.

Request:

- `page_size`
- `page_token`
- optional `provider` filter
- optional `status` filter

Response:

- `provider_grants[]` with metadata only
- `next_page_token`

## `RevokeProviderGrant`

Revokes one owned grant.

Request:

- `provider_grant_id`

Response:

- revoked `provider_grant` metadata

Behavior:

- Marks grant as `revoked`.
- Attempts provider-side revocation via adapter when available.
- Preserves grant history metadata.

## Agent compatibility rule

Provider grants must preserve Phase 1 credential-backed agents.

Compatibility contract:

- Existing agents bound to `credential_id` remain valid.
- Agent auth reference supports exactly one of:
  - `credential_id`
  - `provider_grant_id`
- Validation must enforce provider match and active status for whichever auth
  reference is configured.

## Grant storage and encryption

- Encrypt token material before persistence with AI service encryption key.
- Store only ciphertext + non-secret metadata in DB rows.
- Avoid storing raw OAuth `state`; store state hash and sealed verifier.
- Use UTC timestamps for lifecycle fields.

## Grant authorization and invocation contract

- Caller identity comes from trusted server metadata/auth context.
- All grant operations are owner-scoped by default.
- Not-found vs forbidden behavior should avoid cross-tenant enumeration.
- No client-supplied owner identifiers are accepted.

On invocation requiring a grant:

1. Resolve grant by ID.
2. Verify owner access and `active`-equivalent status.
3. If token is near expiry or unhealthy, attempt refresh according to provider policy.
4. Persist updated token ciphertext and timestamps.
5. Record non-secret refresh/usage metadata.

Failure handling:

- Refresh failures update status (`refresh_failed` or `expired`) and return typed
  service errors.
- Invocation never returns provider tokens.

## Grant security invariants

1. OAuth state/PKCE checks are mandatory for every finish call.
2. Provider tokens and refresh tokens are never returned in API payloads.
3. Secrets are redacted from logs, traces, metrics labels, and error messages.
4. Connect sessions are one-time and short-lived.
5. Revoked grants are never usable for invocation.
6. Ownership checks are server-enforced for all reads/writes.

## Proposed API Surface (Planning)

AI service (`ai.v1`, planned):

- `CredentialService`
  - `CreateCredential`
  - `ListCredentials`
  - `RevokeCredential`
- `ProviderGrantService`
  - `StartProviderConnect`
  - `FinishProviderConnect`
  - `ListProviderGrants`
  - `RevokeProviderGrant`
- `AgentService`
  - `CreateAgent`
  - `UpdateAgent`
  - `ListAgents`
  - `DeleteAgent`
- `AccessRequestService`
  - `CreateAccessRequest`
  - `ListAccessRequests`
  - `ReviewAccessRequest`
- `InvocationService` (internal or protected surface)
  - `InvokeAgent`

Game service additions (planned):

- Campaign-level command/event for AI assignment and unassignment.
- Query/read model fields exposing assigned AI reference state.

## Rollout Plan

## Phase 1: BYO OpenAI API credentials

- Create AI service with credential storage and OpenAI adapter.
- Allow users to create multiple credentials and basic agents.
- No cross-user access requests yet.

## Phase 2: Provider OAuth grants

- Add OAuth connect flow in AI service for provider API grants.
- Support token refresh/revocation lifecycle.
- Durable grant contract is defined in the
  [Provider OAuth Grant Contract](#provider-oauth-grant-contract) section.

## Phase 3: Access-request workflow

- Add request/approval model for owner-managed credentials/agents.
- Add audit events and review tooling.

## Phase 4: Campaign AI assignment

- Add game command/event/read model for campaign AI assignment references.
- Gate assignment to authorized campaign admins with accessible agents.
- Add AI proto contract hardening to model agent auth references as
  `oneof auth_reference` (`credential_id` vs `provider_grant_id`) in
  `Agent`, `CreateAgentRequest`, and `UpdateAgentRequest`.
- Detailed execution spec: [AI Campaign Assignment (Phase 4)](ai-campaign-assignment-phase4.md).

## Phase 5: Runtime GM integration hardening

- Integrate invocation path with chat/game orchestration.
- Capture usage telemetry and failure taxonomy.
- Add deterministic acceptance checks around event persistence.

## Open Questions and Decision Gates

1. Organization/team ownership model:
   - Per-user only first, or user + org ownership from day one?
2. Invocation authority:
   - Should only game call `InvokeAgent`, or allow direct chat/admin paths with
     game-side persistence hooks?
3. Cost controls:
   - Global/project/campaign quotas and failure handling policy.
4. Model pinning:
   - How strict should model/version pinning be for assigned campaign agents?
5. Retention:
   - What invocation artifacts are retained vs excluded for privacy/compliance?

## Acceptance Checks for This Spec

- Boundaries between `auth`, `ai`, and `game` are explicit.
- Resource ownership and storage authority are explicit.
- Campaign assignment is event-sourced and reference-based.
- Replay determinism constraint is explicit for AI-driven gameplay effects.

## Related Docs

- [Architecture](architecture.md)
- [OAuth System](oauth.md)
- [Domain language](domain-language.md)
- [Chat Service](chat-service.md)
- [AI Campaign Assignment (Phase 4)](ai-campaign-assignment-phase4.md)
