# P04: Domain Lifecycle Packages and Typed Invariants

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the AI domain packages to verify that lifecycle transitions, type vocabulary, and invariants live in the correct place and are not leaking into transport, service, or storage code.

Primary scope:

- `internal/services/ai/agent`
- `internal/services/ai/credential`
- `internal/services/ai/providergrant`
- `internal/services/ai/accessrequest`
- `internal/services/ai/secret`
- `internal/services/ai/debugtrace`

## Progress

- [x] (2026-03-23 04:27Z) Reviewed domain types, package comments, transition helpers, and targeted tests across `agent`, `credential`, `providergrant`, `accessrequest`, `secret`, and `debugtrace`.
- [x] (2026-03-23 04:28Z) Verified domain-package baseline with `go test ./internal/services/ai/agent ./internal/services/ai/credential ./internal/services/ai/providergrant ./internal/services/ai/accessrequest ./internal/services/ai/secret`.
- [x] (2026-03-23 04:33Z) Synthesized domain lifecycle findings, typed-vocabulary drift, and the target cleanup order.

## Surprises & Discoveries

- The domain layer is in better shape than the service and transport layers: most lifecycle transitions already live in the correct owning package.
- `agent.AuthReference` is a legitimate domain improvement, but outer layers still project it back into legacy nullable `credential_id` and `provider_grant_id` pairs.
- `providergrant` is the clearest lifecycle package in scope: refresh success/failure semantics are already centralized in the domain instead of spread across services or sqlite.
- `debugtrace` is more of a trace-record vocabulary package than a rich lifecycle package, which is acceptable, but that also means its parse/value helpers currently have no direct package tests.

## Decision Log

- Decision: Keep the current domain package split and treat P04 as a boundary-cleanup pass, not a package-reorganization pass.
  Rationale: `agent`, `credential`, `providergrant`, and `accessrequest` each have a coherent owner concept and already expose meaningful lifecycle helpers. The cleanup target is vocabulary drift and outer-layer duplication, not moving domain types around.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

P04 is complete for planning purposes. The main conclusion is that the domain layer should remain the authority for AI lifecycle rules, but several outer layers still drag storage- and transport-shaped representations around those domain types. The clean path is to preserve the existing domain package split, tighten vocabulary/doc alignment, finish the `AuthReference` cutover later in storage/transport passes, and decide whether `accessrequest` needs a cleaner revoke vocabulary instead of overloading review metadata.

## Context and Orientation

Use:

- `docs/reference/ai-service-lifecycle-terms.md`
- `docs/architecture/foundations/domain-language.md`
- package `doc.go` files in each scoped domain package

## Plan of Work

Inspect:

- transition purity and invariants
- typed selectors and lifecycle vocabulary
- whether domain packages expose transport/storage concerns
- status and auth-reference modeling
- secret-handling boundary placement
- debug trace ownership and value-type quality

## Current Findings

### F01: `agent` package comments drift from the actual lifecycle vocabulary

Category: missing best practice, contributor-clarity risk

Evidence:

- `internal/services/ai/agent/doc.go:3-6` says the agent lifecycle includes `active/archived`.
- `internal/services/ai/agent/agent.go:21-24` defines only `StatusActive`.

Impact:

- New contributors get a false picture of the lifecycle surface before they even reach the code.
- Later refactor passes cannot tell whether `archived` is intentionally missing, intentionally removed, or merely undocumented drift.

Refactor direction:

- Treat the code as authoritative for now and fix the package docs first.
- If archival is intended, add it as a real domain state with transitions in a later implementation pass instead of leaving it implied in comments.

### F02: `agent.AuthReference` owns the right invariant, but outer layers still rely on legacy nullable ID pairs

Category: refactor candidate, maintainability risk

Evidence:

- `internal/services/ai/agent/auth_reference.go:15-75` correctly centralizes typed auth-reference normalization and exclusivity rules.
- `internal/services/ai/service/agent.go:107`, `:164`, and `:305` still rebuild auth references from `CredentialID` and `ProviderGrantID` inputs.
- `internal/services/ai/storage/sqlite/store_agents.go:44-69` persists the domain value back into `credential_id` and `provider_grant_id`.
- `internal/services/ai/storage/sqlite/store_agents.go:267` reconstructs the typed value through `agent.AuthReferenceFromIDs(...)`.
- `internal/services/ai/api/grpc/ai/proto_helpers.go:156-157` projects the domain value back into the proto pair shape.

Impact:

- The domain package is the correct authority, but the system still pays a translation tax in service, transport, and sqlite.
- Contributor learning cost stays high because the codebase carries both the typed domain term and the older nullable-pair vocabulary.
- Later refactors risk reintroducing duplicated exclusivity rules at the edges.

Refactor direction:

- Keep `agent.AuthReference` as the sole domain authority.
- In later passes, treat `credential_id` and `provider_grant_id` as compatibility projections to delete from outer contracts where possible.
- Prefer transport/storage contracts that carry the typed selector directly instead of forcing repeated pair reconstruction.

### F03: `credential.Credential` mixes creation-time plaintext and persisted ciphertext concerns in one domain type

Category: missing best practice, maintainability and testability risk

Evidence:

- `internal/services/ai/credential/credential.go:39-55` includes both `Secret` and `SecretCiphertext`.
- The package comment in `internal/services/ai/credential/credential.go:1-5` says encryption belongs to higher layers, but the persisted ciphertext still lives on the domain struct.

Impact:

- The package carries two different lifecycle moments in one type: user input before sealing and storage state after sealing.
- Tests and contributors have to infer which fields are meaningful in which phase.
- The boundary between domain validation and storage persistence is less crisp than it should be for secret-bearing types.

Refactor direction:

- Decide explicitly whether the project wants one dual-purpose domain record or separate create-time and persisted shapes.
- If the dual-purpose shape stays, document the allowed field combinations more explicitly in the package comment and constructors.
- If the project prefers cleaner lifecycle separation, split plaintext create input from persisted credential state in a later refactor pass.

### F04: `accessrequest` overloads review metadata for revocation semantics

Category: anti-pattern

Evidence:

- `internal/services/ai/accessrequest/accessrequest.go:92-97` only defines `ReviewerUserID`, `ReviewNote`, and `ReviewedAt` as decision metadata.
- `internal/services/ai/accessrequest/accessrequest.go:257-283` implements `Revoke(...)` by reusing `ReviewerUserID` and `ReviewNote`, and it does not set a dedicated revoke timestamp field.

Impact:

- The lifecycle vocabulary is partly storage-shaped instead of domain-clean.
- Review and revoke are distinct business actions, but the record shape makes them look like the same event family.
- Transport/storage layers have less precise fields to expose if revocation behavior grows beyond the current narrow workflow.

Refactor direction:

- Add explicit revoke metadata if revocation is intended to remain a first-class lifecycle step.
- Keep `Review(...)` and `Revoke(...)` as separate domain transitions, but give them state fields that match their actual meaning.

### F05: `debugtrace` has stable value vocabulary, but no package-local tests protect it

Category: missing best practice, testability gap

Evidence:

- `internal/services/ai/debugtrace/types.go:10-60` contains normalization helpers for `Status` and `EntryKind`.
- `internal/services/ai/debugtrace` currently contains only `doc.go` and `types.go`; there are no package tests.

Impact:

- The package is small, but it still defines persisted vocabulary used across storage and transport.
- Parse-helper regressions would be caught only indirectly through higher-level tests.

Refactor direction:

- Add a small package-local table test for `ParseStatus` and `ParseEntryKind`.
- Keep the package intentionally record-shaped unless a later pass finds real lifecycle behavior that belongs here.

### F06: `providergrant` is the healthiest lifecycle package in scope and should remain the single owner of refresh-state transitions

Category: best practice already employed, preserve as-is

Evidence:

- `internal/services/ai/providergrant/providergrant.go:188-229` owns `RecordRefreshSuccess(...)` and `RecordRefreshFailure(...)`.
- Service code consumes those helpers instead of mutating refresh fields ad hoc.

Impact:

- Refresh semantics are auditable and package-local.
- The package provides the right model for other lifecycle-heavy AI domains: typed statuses plus explicit transitions.

Refactor direction:

- Preserve this ownership model in later service/provider passes.
- Delete any future direct refresh-field mutation outside `providergrant`.

## Concrete Steps

1. Read package comments and exported types first.
2. Trace create/update/revoke/refresh flows back to the owning package.
3. Record invariants that are duplicated outside the domain.
4. Propose deletions of duplicate validation logic from outer layers.

## Target Domain Shape

Keep the current package ownership:

- `agent` owns agent lifecycle state and typed auth-reference selection.
- `credential` owns plaintext input validation and credential lifecycle state.
- `providergrant` owns OAuth grant lifecycle and refresh-state transitions.
- `accessrequest` owns request/review/revoke workflow state.
- `secret` remains a sealing boundary utility package.
- `debugtrace` remains a trace-record vocabulary package unless later passes discover real lifecycle behavior that belongs there.

Clean up the outer contracts and docs around that ownership:

1. Align package comments and reference docs to the live domain vocabulary.
   - fix `agent` lifecycle wording first
   - add explicit wording for any dual-purpose secret-bearing fields that remain
2. Treat `agent.AuthReference` as the authoritative selector and push nullable ID pairs outward until they can be removed.
   - later transport/storage passes should prefer a typed selector over pair reconstruction
3. Keep provider-grant refresh transitions centralized in `providergrant`.
   - later passes should move policy around refresh timing, not state mutation ownership
4. Decide whether access-request revocation deserves its own metadata vocabulary.
   - if yes, add explicit revoke fields and stop overloading review metadata
   - if no, document clearly that revoke is represented as a second owner decision using the same metadata fields
5. Add focused tests for small domain vocabulary packages.
   - `debugtrace` should have direct tests once implementation changes begin

## Validation and Acceptance

- `go test ./internal/services/ai/agent ./internal/services/ai/credential ./internal/services/ai/providergrant ./internal/services/ai/accessrequest ./internal/services/ai/secret`
- `go test ./internal/services/ai/...`

Acceptance:

- each lifecycle rule has a single owning package
- typed vocabulary drift is identified or cleared
- any exported type changes are recorded

## Idempotence and Recovery

- Treat domain duplication outside the owning package as a candidate for deletion unless a strong counterexample appears.

## Artifacts and Notes

- Capture any domain term that needs promotion into docs/reference.
- `agent` package comments and `docs/reference/ai-service-lifecycle-terms.md` should stay aligned once auth-reference cutover work starts.

## Interfaces and Dependencies

Track any proposed changes to:

- `agent.AuthReference`
- status enums/types
- lifecycle helpers and constructors
- `credential.Credential` secret-field shape
- `accessrequest.AccessRequest` review/revoke metadata fields

## Cutover Order

1. Fix package-comment and reference-doc drift so contributors see the correct lifecycle vocabulary.
2. Decide the `accessrequest` revoke metadata shape before transport or storage contracts grow around the overloaded fields.
3. In later transport/storage passes, continue the `agent.AuthReference` cutover and delete nullable pair reconstruction where breaking changes are acceptable.
4. Add package-local `debugtrace` tests once that package is touched by implementation work.
