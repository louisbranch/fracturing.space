---
title: "Web Capability Parity"
parent: "Project"
nav_order: 12
---

# Web Capability Parity Spec

## Purpose

Define an initial path for evolving the web service into the primary user-facing site for campaign planning while preserving strict user scoping and reusing proven admin patterns.

This spec is based on the current code surfaces as of 2026-02-15.

## Current State Snapshot

### Admin service (`internal/services/admin`)

Current admin UI supports broad operational coverage, including:

- Campaign listing, create flow, and campaign detail.
- Campaign tabs for sessions, characters, participants, invites, and event log.
- Session and character detail pages.
- User tools, including impersonation and pending-invites view.
- Systems/catalog/icons/scenarios operational pages.

### Web service (`internal/services/web`)

Current web UI supports:

- Public landing page.
- OAuth login/callback/logout.
- Passkey registration/login.
- Magic-link consume flow.
- One participant-gated route: `/campaigns/{id}` rendering a phase-1 chat shell.

Current web does not yet provide:

- User-scoped campaign index.
- Session/participant/character/invite management pages.
- Role-aware management actions in UI.

## Product Goals

- Make web the default end-user surface for campaign planning.
- Keep all data and actions scoped to the signed-in user.
- Support both participant view and campaign manager/owner actions.
- Reuse admin patterns/components where practical to reduce divergence.

## Non-goals (Initial)

- Porting admin-only operational surfaces (systems, catalog, icons, scenarios, full user admin).
- Replacing gRPC/MCP APIs.
- Building a full VTT/chat/media platform in this phase.

## User and Authorization Model

### Identity source

- Web session remains based on OAuth login (`fs_session` + domain-scoped `fs_token`).
- Request-scoped user identity should be derived from token introspection and propagated internally.

### Scope model

- Every campaign page must require campaign membership for the current user.
- Manager/owner actions must require participant-level access checks.
- Participant/member users should only see and perform allowed actions.

### Metadata propagation to game gRPC

Web must set metadata explicitly when calling game APIs:

- `x-fracturing-space-user-id` for user-scoped operations (for example campaign creation, pending invites for user, invite claim).
- `x-fracturing-space-participant-id` for campaign-scoped management operations.

### Current backend scoping status

Current game APIs have mixed scoping enforcement:

- `ListCampaigns` is user-scoped when `x-fracturing-space-user-id` is present.
- `ListPendingInvitesForUser` and `ClaimInvite` require `x-fracturing-space-user-id`.
- Invite management APIs (`ListInvites`, `CreateInvite`, `RevokeInvite`) require `x-fracturing-space-participant-id` with manager/owner access.
- Most campaign read APIs (`GetCampaign`, `ListSessions`, `ListParticipants`, `ListCharacters`) are not yet consistently user-scoped by metadata and still rely on caller-side route guards.

Until read-side authorization is fully centralized in game service, web must keep campaign membership checks at the web route layer.

## Proposed Web Information Architecture

- `/` public landing/login entry.
- `/app` authenticated shell.
- `/app/campaigns` user-scoped campaign list.
- `/app/campaigns/{campaignID}` campaign overview.
- `/app/campaigns/{campaignID}/sessions` sessions list/detail links.
- `/app/campaigns/{campaignID}/participants` participants list.
- `/app/campaigns/{campaignID}/characters` characters list/detail links.
- `/app/campaigns/{campaignID}/invites` invite management (manager/owner).
- `/app/invites` pending invites for current user + claim flow.

## Capability Matrix

| Capability | Admin today | Web today | Web target |
|---|---|---|---|
| Campaign list | Yes | No | Yes (user-scoped) |
| Campaign create | Yes | No | Yes (signed-in user only) |
| Campaign detail overview | Yes | Limited (chat shell) | Yes |
| Session list/detail | Yes | No | Yes |
| Session start/end | Partial via API/ops | No | Yes (manager/owner) |
| Participant list | Yes | No | Yes |
| Participant manage (role/access/controller) | Via APIs/admin tools | No | Yes (manager/owner) |
| Character list/detail | Yes | No | Yes |
| Character create/update/control assignment | Via APIs/admin tools | No | Yes (role-aware) |
| Invite list/create/revoke | Yes | No | Yes (manager/owner) |
| Pending invites for current user | Yes (impersonation path) | No | Yes |
| Invite claim | API available, not user UX | No | Yes |

## Phased Delivery Plan

### Phase 1: Foundation and Auth Context

- Introduce authenticated `/app` shell in web.
- Add reusable request auth context (user id, token state, locale).
- Add campaign membership resolver used by all campaign routes.
- Keep current `/campaigns/{id}` chat route working while moving toward `/app/*`.

### Phase 2: Read Parity (User-Scoped)

- Implement user-scoped campaign list and campaign overview pages.
- Add sessions, participants, and characters read pages under campaign scope.
- Add route-level guardrails to reject any campaign access outside membership.

### Phase 3: Management Actions (Role-Aware)

- Add campaign creation from web using authenticated user context.
- Add manager/owner actions for participants, invites, and characters.
- Add session start/end controls with clear role gating in UI and handler checks.

### Phase 4: Invite Lifecycle UX

- Add “My Invites” page backed by `ListPendingInvitesForUser`.
- Add claim-invite flow (`IssueJoinGrant` + `ClaimInvite`) in web UX.
- Surface post-claim routing into campaign workspace.

### Phase 5: Hardening and Convergence

- Reduce temporary web-layer-only guards as game-layer scoping matures.
- Remove legacy route duplication once `/app/*` is stable.
- Add parity smoke coverage for core user journeys.

## Recommended Starting Slice

Start with a thin vertical slice that is user-visible, low-risk, and validates shared auth plumbing:

1. `/app/campaigns` user-scoped list (read-only) using current membership filtering safeguards.
2. `/app/campaigns/{campaignID}` overview read page (reuse existing campaign access checker).
3. Shared metadata/context wiring used by both routes (`x-fracturing-space-user-id` path).
4. Route-level tests for unauthenticated redirect, unauthorized campaign rejection, and happy-path render.

Why this first: it exercises auth/session/context boundaries, establishes `/app` navigation, and avoids role-sensitive write actions until read parity is stable.

## Route/API Mapping (Initial)

| Web capability | Primary API calls | Required metadata | Admin reference |
|---|---|---|---|
| `/app/campaigns` list | `CampaignService.ListCampaigns` | `x-fracturing-space-user-id` | `handleCampaignsTable` |
| `/app/campaigns/{id}` overview | `CampaignService.GetCampaign` | route-level membership guard | `handleCampaignDetail` |
| `/app/campaigns/{id}/sessions` | `SessionService.ListSessions` | route-level membership guard | `handleSessionsTable` |
| `/app/campaigns/{id}/participants` | `ParticipantService.ListParticipants` | route-level membership guard | `handleParticipantsTable` |
| `/app/campaigns/{id}/characters` | `CharacterService.ListCharacters` | route-level membership guard | `handleCharactersTable` |
| `/app/campaigns/{id}/invites` | `InviteService.ListInvites` | `x-fracturing-space-participant-id` | `handleInvitesTable` |
| `/app/invites` | `InviteService.ListPendingInvitesForUser` | `x-fracturing-space-user-id` | `listPendingInvitesForUser` |
| invite claim | `AuthService.IssueJoinGrant` + `InviteService.ClaimInvite` | `x-fracturing-space-user-id` | (admin has API helpers only) |

## Execution Plan (PR Order)

1. PR1: normalize `/app/campaigns` and `/app/campaigns/{id}` as canonical routes; add redirects from legacy entry points where needed.
2. PR2: add read-only sessions/participants/characters pages under `/app/campaigns/{id}/*` with shared membership guard helper.
3. PR3: add `/app/invites` pending-invites page and claim flow UI wiring (`IssueJoinGrant` + `ClaimInvite`).
4. PR4: add manager/owner invite management page under campaign scope with participant-id metadata propagation.
5. PR5: add parity smoke tests for signed-in user journeys and remove redundant legacy route logic after cutover confidence.

## Legacy Route Cutover Decision (2026-02-16)

- Legacy `/campaigns/{id}` has been removed.
- Canonical campaign workspace entry is `/app/campaigns/{id}`.
- Requests to `/campaigns/{id}` now return `404 Not Found`.
- Existing links should be migrated to `/app/campaigns/{id}`.

Rationale: route duplication no longer provides product value and keeping one canonical URL simplifies parity work and testing.

## Implementation Update (2026-02-16)

- Campaign invite management in web is now role-gated for manager/owner at both handler and UI levels on `/app/campaigns/{campaignID}/invites`.
- Members are rejected server-side with `403 Forbidden` for invite list/create/revoke actions.
- Invite create/revoke controls are hidden from non-manager/owner views.
- Web now maps common invite gRPC failures to user-facing HTTP status codes (`400`, `401`, `403`, `404`, `409`, `503`) instead of always returning `502`.
- Invite handlers now resolve participant identity/access once per request path to avoid duplicate participant-list lookups.

## Shared Package Opportunities

The codebase already has `internal/services/shared/templates` for shared icon components. Extend this shared approach incrementally.

### 1) Shared auth utilities split by boundary (decided)

Use two shared locations with clear responsibilities:

- `internal/platform/requestctx`: request-scoped identity context helpers (for example `WithUserID`, `UserIDFromContext`).
- `internal/services/shared/authctx`: auth-service contract client for HTTP token introspection.

Why: context primitives are platform-level concerns, while introspection is a service-contract integration concern.

### 2) Shared i18n HTTP helpers

Create a shared package (for example `internal/services/shared/i18nhttp`) for:

- `ResolveTag`, `SetLanguageCookie`, `LangParam`, `LangCookieName`.

Why: admin and web i18n packages are functionally identical.

### 3) Shared template utility primitives

Create or extend shared template utilities for:

- `Localizer` interface + `T()` helper.
- Language option helpers used in both template sets.
- Common page context pieces where semantics match.

Why: admin and web template utility files currently duplicate logic with only package-local naming differences.

### 4) Shared gRPC metadata builder for user/participant context

Create a shared helper (for example `internal/services/shared/grpcauthctx`) to attach user/participant metadata consistently.

Why: avoids subtle divergence in header wiring and role checks across handlers.

## Backend/API Follow-ups Needed for True Web-First Scoping

- Add user-scoped campaign listing/query API (or equivalent filter contract).
- Ensure read endpoints needed by web are enforceable by user/participant scope at game service layer.
- Confirm write endpoints that will be exposed in web enforce role/permission server-side, not only in UI.

## Acceptance Criteria for Initial Implementation

- Authenticated user can view only campaigns where they are a participant.
- Authenticated user can navigate sessions/participants/characters for permitted campaigns.
- Manager/owner-only actions are hidden and rejected server-side when unauthorized.
- User can see pending invites and claim an invite from web UX.
- Shared auth utilities are split across platform context helpers and service-shared introspection helpers.

## Risks and Mitigations

- Risk: exposing web routes before backend scoping hardening leaks data.
  Mitigation: enforce membership checks in web handler layer for every campaign-scoped route and prioritize game API scoping follow-ups.

- Risk: admin and web diverge further while parity work proceeds.
  Mitigation: extract shared primitives early (auth/i18n/context) before adding large new web surfaces.

- Risk: role resolution errors for manager/owner actions.
  Mitigation: resolve participant identity per campaign and centralize authorization checks in one handler helper.

## Out of Scope for This Spec

- Detailed UI visual design.
- Frontend framework migration decisions.
- Deployment topology changes.
