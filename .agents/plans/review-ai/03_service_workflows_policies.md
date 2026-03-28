# P03: Service Workflow Boundaries, Shared Policies, and Error Taxonomy

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the AI service layer to determine whether workflow services own the right responsibilities, share policy cleanly, and expose maintainable/testable boundaries for authorization, token resolution, usage guards, and error handling.

Primary scope:

- `internal/services/ai/service`

## Progress

- [x] (2026-03-23 04:18Z) Reviewed workflow services, shared helpers, and service tests across `internal/services/ai/service`.
- [x] (2026-03-23 04:18Z) Verified service baseline with `go test ./internal/services/ai/service`.
- [x] (2026-03-23 04:21Z) Synthesized the target helper/policy split and cutover order for the service layer.

## Surprises & Discoveries

- The package-level structure matches the docs: workflow services are easy to locate.
- The main service-layer complexity is not the number of workflow services; it is the breadth of the shared helpers that several workflows depend on.
- `AccessRequestService` still carries production `SetClock` and `SetIDGenerator` methods even though constructor injection already exists for both dependencies.

## Decision Log

- Decision: Keep workflow-family service structs as the primary service-layer organizing unit.
  Rationale: The current top-level service split is understandable; the cleanup target is the helper and policy boundary beneath those services, not collapsing everything into a generic application service.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

P03 is stable enough to treat as complete for planning purposes. The service-layer issue is not the top-level workflow split; it is that several “helper” components are actually multi-role coordinators. The clean path is to keep workflow services, but split shared runtime auth, binding-usage policy, and reusable auth-reference validation into smaller named seams, while deleting production test-setter escape hatches.

## Context and Orientation

Read before recording findings:

- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-lifecycle-terms.md`
- `internal/services/ai/service/doc.go`
- `internal/services/ai/service/errors.go`
- `internal/services/ai/service/deps.go`

## Plan of Work

Inspect:

- one-service-per-workflow boundary quality
- shared helper ownership
- constructor/config noise
- repeated ownership/auth checks
- error taxonomy and transport leakage
- test seam quality for clocks, IDs, provider adapters, and stores
- whether workflow services are too storage-shaped or too transport-shaped

## Current Findings

### F01: `AuthTokenResolver` is broader than its name and acts as a mutating provider-grant lifecycle coordinator

Category: anti-pattern, maintainability risk

Evidence:

- `auth_token_resolver.go` handles:
  - credential lookup and decryption
  - provider-grant lookup and usability checks
  - proactive refresh policy
  - provider refresh HTTP calls
  - provider-grant lifecycle state writes on success/failure
- The helper name suggests a read-only token lookup seam, but it performs durable mutation through `PutProviderGrant`.

Impact:

- Callers cannot tell from the name whether resolution is read-only or may write state.
- Refresh lifecycle policy is hidden inside a helper that many workflows will treat as infrastructure.
- The dependency surface is broad: stores, provider adapters, sealer, and clock.

Refactor direction:

- Split the concern into:
  - an auth-reference material resolver for read/open operations
  - a provider-grant runtime/refresher component that owns refresh policy and persistence
- Make mutation explicit in the helper naming and constructor boundaries.

### F02: `AgentService` owns too much auth-reference and model-availability policy inline

Category: anti-pattern

Evidence:

- `agent.go` contains:
  - `validateAgentAuthReferenceForProvider`
  - `validateProviderModelAvailable`
  - `GetAuthState`
- These helpers directly query credential/grant stores and provider model adapters in addition to the workflow methods already coordinating create/update/list behavior.

Impact:

- The agent workflow struct is carrying both workflow orchestration and reusable auth-reference policy.
- Auth-reference semantics are partly centralized in `AuthTokenResolver` and partly reimplemented in `AgentService`.
- Future changes to runtime auth readiness or model validation will likely require touching multiple helpers across the package.

Refactor direction:

- Extract a narrower auth-reference policy component for:
  - validating owner/provider usability
  - deriving non-mutating auth readiness
  - checking model availability for a chosen auth reference
- Keep `AgentService` focused on create/update/delete/list workflow orchestration.

### F03: `UsageGuard` is a policy object, a store scanner, and a cross-service usage reader at once

Category: anti-pattern, testability risk

Evidence:

- `usage_guard.go` performs owner-agent pagination through `AgentStore`.
- It also calls `gameCampaignAIClient.GetCampaignAIBindingUsage`.
- It is used as the policy gate for credential revoke, provider-grant revoke, agent update, and agent delete.

Impact:

- The name “guard” undersells how much work and I/O it performs.
- Tests and callers have to reason about both store pagination behavior and remote game-service usage semantics through one helper.
- The policy boundary is harder to evolve because local matching and remote usage checks are fused.

Refactor direction:

- Split usage/binding reads from the policy decision.
- Suggested shape:
  - one binding-usage reader that knows how to answer “how many active campaigns use this agent?”
  - one auth-reference usage reader that maps credential/provider-grant usage through agent listings
  - one small policy helper that turns those answers into failed-precondition decisions

### F04: `AccessRequestService` still carries production test-setter methods despite constructor injection

Category: anti-pattern

Evidence:

- `access_request.go:299-307` exposes `SetClock` and `SetIDGenerator`.
- The same service already accepts `Clock` and `IDGenerator` through `AccessRequestServiceConfig`.
- No service tests in the package use the setter methods.

Impact:

- Production code is carrying an extra mutation surface purely for tests.
- The package already has the correct injection mechanism, so the setters create two ways to do the same thing.
- This weakens the service-layer stance that dependencies should be fixed at construction.

Refactor direction:

- Delete `SetClock` and `SetIDGenerator`.
- Standardize on constructor injection for all service tests and transport test builders.

### F05: Shared helper error behavior is not fully uniform inside the service package

Category: missing best practice

Evidence:

- Public workflow methods generally return `service.Error` values via `Errorf` and `Wrapf`.
- Internal helper paths such as `AuthTokenResolver.refreshProviderGrant` return raw `fmt.Errorf` values and rely on outer layers to remap or collapse them.
- Constructors also use raw `fmt.Errorf`, which is acceptable at composition time, but the helper inconsistency makes service-internal policy harder to audit.

Impact:

- The service package does not have one obvious internal rule for when helpers must return typed service errors vs raw Go errors.
- Error-kind reasoning becomes more indirect in complex helper chains like provider-grant refresh.

Refactor direction:

- Keep raw constructor errors for composition-time validation.
- For runtime helpers that are part of workflow execution, prefer one internal convention:
  either return typed service errors directly, or keep raw errors strictly private behind one outer wrapper boundary.
- Make that boundary explicit in helper names and comments.

## Concrete Steps

1. Map each service struct and its direct dependencies.
2. Note which shared helpers are healthy cross-cutting policy and which are catch-all spillover.
3. Compare the documented service-layer contract to actual implementations.
4. Record concrete refactor slices, including likely deletions or service splits.
5. Convert findings into a target helper/policy shape and cutover order.

## Target Service-Layer Shape

Keep the current workflow-family service split:

- `CredentialService`
- `ProviderGrantService`
- `AgentService`
- `InvocationService`
- `AccessRequestService`
- `CampaignOrchestrationService`
- `CampaignDebugService`

Refactor shared helpers into narrower policy/runtime components:

1. Split runtime auth resolution from provider-grant refresh lifecycle.
   - one component opens current auth material for a validated auth reference
   - one component owns provider-grant refresh policy and persistence
2. Extract reusable auth-reference policy from `AgentService`.
   - validate owner/provider usability
   - compute non-mutating auth readiness
   - validate model availability for a selected auth reference
3. Split usage/binding reads from “guard” decisions.
   - a reader layer answers usage questions
   - a small policy layer converts those answers into precondition errors
4. Keep audit writing explicit, but avoid test-only mutation APIs on services.
   - constructor injection remains the only dependency override mechanism
5. Standardize runtime helper error behavior.
   - constructors may keep raw composition-time errors
   - runtime helper paths should have one obvious boundary where raw errors become typed service errors

## Cutover Order

1. Delete `AccessRequestService` test setters and normalize all tests/builders to constructor injection.
2. Extract shared auth-reference policy from `AgentService` without changing workflow method signatures.
3. Split `AuthTokenResolver` into explicit read-vs-refresh components and switch `InvocationService` and `CampaignOrchestrationService` to the new seams.
4. Split `UsageGuard` into usage readers plus a thin policy layer, then rewire credential/provider-grant/agent workflows.
5. Normalize helper runtime error conventions once the helper boundaries are smaller and easier to audit.

## Validation and Acceptance

- `go test ./internal/services/ai/service`
- `go test ./internal/services/ai/...`

Acceptance:

- target ownership for each workflow family is explicit
- shared-policy placement is explicit
- error interface impact is recorded
- helper boundaries are specific enough to refactor without broad service churn
- cutover order is explicit enough to implement in coherent slices

## Idempotence and Recovery

- If later passes discover lifecycle rules living in the wrong layer, update this file rather than forcing later passes to compensate.

## Artifacts and Notes

- Capture any service helper that deserves promotion to its own package or deletion outright.

## Interfaces and Dependencies

Track any proposed changes to:

- `service.Error` and `ErrorKind`
- workflow service configs/constructors
- helper seams such as token resolution and usage guarding
