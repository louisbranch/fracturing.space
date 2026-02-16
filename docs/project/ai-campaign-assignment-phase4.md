---
title: "AI Campaign Assignment (Phase 4)"
parent: "Project"
nav_order: 16
---

# Game Service Phase 4: Campaign AI Assignment

## Purpose

Define an implementation-ready specification for assigning one AI agent to a
campaign GM role while preserving game-service event authority and AI-service
credential boundaries.

Phase 4 goal:

- Campaign owners/managers can assign or clear an AI agent reference for a
  campaign.
- Assignment state is event-sourced in game service.
- Assignment validation depends on AI service accessibility checks, not direct
  token access.

## Scope and Non-goals

In scope:

- Campaign assignment command/event/read model.
- Permission checks for assignment actions.
- AI dependency contract for validating one candidate agent ID.

Out of scope:

- GM behavior orchestration/runtime prompting (Phase 5).
- Web page UX details.
- AI provider token lifecycle changes.

## Service Boundary

- Game service owns campaign assignment state and audit/event history.
- AI service owns agent accessibility and provider credential security.
- Game service stores only `agent_id` references and assignment metadata.
- Game service never stores provider secrets.

## Domain Model

## CampaignAIAssignment

Event-sourced campaign fact with these fields:

- `campaign_id`
- `agent_id` (empty when unassigned)
- `assigned_by_participant_id`
- `assigned_at`
- `unassigned_at` (optional)
- `status` (`assigned` or `unassigned`)

The assignment is a campaign reference, not an ownership transfer.

## API Surface (Planning)

Game `CampaignService` additions:

- `AssignCampaignAI`
  - request: `campaign_id`, `agent_id`
  - response: campaign with assignment fields
- `UnassignCampaignAI`
  - request: `campaign_id`
  - response: campaign with assignment fields

Read surface:

- campaign query responses include assignment reference fields (for example
  `ai_agent_id` and assignment timestamps).

## Dependency Contract (AI service)

Game service needs point-in-time accessibility validation for a single agent ID.

AI `AgentService` dependency:

- `GetAccessibleAgent`
  - request: `agent_id`
  - response: `agent` when caller can access it
  - returns not found when caller cannot access or agent does not exist

Rationale: assignment checks are by explicit ID; list pagination is not a
stable or efficient authorization primitive for this flow.

Phase 4 dependency hardening:

- AI proto should evolve agent auth reference fields to `oneof auth_reference`
  across `Agent`, `CreateAgentRequest`, and `UpdateAgentRequest`.
- Purpose: encode the existing runtime invariant ("exactly one of
  credential_id or provider_grant_id") at the API contract layer, reducing
  ambiguous client payloads before campaign assignment integration scales.

## Authorization Rules

- Only campaign participants with owner/manager access can assign/unassign.
- Assignment requires that the acting user can access the referenced AI agent
  through AI service policy (owner or approved delegated invoke access).
- Unauthorized assignment attempts must not leak cross-tenant agent existence.

## Event and Projection Requirements

- Assignment/unassignment emits campaign-domain events.
- Projection updates campaign read model with assignment reference fields.
- Replay must reconstruct assignment deterministically from events.

## Security Invariants

1. Game service persists references only, never secrets.
2. AI accessibility is validated server-side at assignment time.
3. Assignment events include actor metadata for auditing.
4. Not-found style masking is used for inaccessible foreign agents.
5. Runtime invocation uses assigned reference but must re-check access.

## Error Taxonomy

Canonical categories:

- `invalid_argument`: missing campaign/agent IDs.
- `permission_denied`: actor lacks campaign management rights.
- `not_found`: campaign not found, inaccessible agent, or inaccessible resource.
- `failed_precondition`: campaign status blocks assignment.
- `internal`: storage/domain/dependency call failures.
- `unavailable`: transient AI dependency outage.

## Phase 4 Acceptance Checks

- Owner/manager can assign an accessible agent to campaign.
- Owner/manager can clear assignment.
- Member/non-participant cannot mutate assignment.
- Inaccessible agent IDs cannot be assigned.
- `GetCampaign`/`ListCampaigns` reflect current assignment reference.
- Replay reconstructs assignment state without AI API calls.

## Implementation Notes

- Prefer extending existing campaign domain update/event paths to minimize new
  projection branches.
- Keep assignment validation close to command handling to avoid stale check/use
  windows.
- Add integration tests covering game->AI dependency behavior under allow/deny.
