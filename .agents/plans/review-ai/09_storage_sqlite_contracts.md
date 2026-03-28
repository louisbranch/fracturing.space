# P09: Storage Contracts, SQLite Adapter Layout, Migrations, and Pagination/Filter Contracts

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the AI storage interfaces and SQLite implementation to determine whether repository seams are small, domain-aligned, easy to fake, and clear for contributors who need to change persistence behavior safely.

Primary scope:

- `internal/services/ai/storage`
- `internal/services/ai/storage/sqlite`
- `internal/services/ai/storage/sqlite/migrations`

## Progress

- [x] (2026-03-23 03:59Z) Reviewed repository interfaces, record shapes, sqlite helper organization, and representative service/test callers.
- [x] (2026-03-23 04:32Z) Recorded findings and proposed target storage seam cleanup.
- [x] (2026-03-23 04:35Z) Validation complete: `go test ./internal/services/ai/storage/sqlite` passed.
- [x] (2026-03-23 04:36Z) Validation complete: `go test ./internal/services/ai/...` passed.

## Surprises & Discoveries

- The storage layer is split more cleanly by file than by contract vocabulary. Domain aggregates scan directly into domain structs, but several support workflows still expose storage-owned record types.
- The architecture doc says there are no separate storage record types, but `storage.go` still defines `AuditEventRecord`, `CampaignArtifactRecord`, and `ProviderConnectSessionRecord`.
- `storage.Store` exists as an aggregate “all interfaces” umbrella, but the codebase does not appear to use it.
- The sqlite adapter is organized by aggregate file, which is good, but common validation and pagination mechanics are still copied by hand in most methods.
- The auth-reference cleanup thread from P04 is still blocked at the schema edge: sqlite persists agent auth as `credential_id` plus `provider_grant_id`.
- Migration safety is mostly implicit. Fresh-schema coverage comes from store tests opening a temp DB, but the migration package itself has almost no direct regression coverage.

## Decision Log

- Decision: Preserve direct sqlite-to-domain scanning for the real domain aggregates.
  Rationale: `credential`, `agent`, `providergrant`, `accessrequest`, and `debugtrace` are easier to reason about because the adapter reconstructs their canonical domain types directly instead of introducing extra record DTOs.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat storage-owned record contracts as shrink targets unless they represent a stable domain concept with its own vocabulary.
  Rationale: The current mix of domain types and storage-owned record types is difficult for contributors to predict and makes test doubles more storage-shaped than behavior-shaped.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat transaction support as a missing storage capability, not as an optional optimization.
  Rationale: Some workflows already perform coupled writes across aggregates without a transaction seam. That is a contract gap, not just sqlite implementation detail.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

Findings are grouped by review goal and classified as missing best practice, anti-pattern, or refactor candidate.

### Maintainability

1. Anti-pattern: the storage contract layer mixes domain-shaped repositories with storage-owned record DTOs.
   Evidence:
   - `storage.go` returns domain types for credentials, agents, provider grants, access requests, and debug traces.
   - the same package also defines `AuditEventRecord`, `CampaignArtifactRecord`, and `ProviderConnectSessionRecord`.
   - `docs/architecture/platform/ai-service-architecture.md` says there are no separate storage record types.
   Why it matters:
   - Contributors cannot predict whether a new persistence seam should return a domain type or a storage-specific record shape.
   Refactor direction:
   - Either create real domain/support packages for audit events, artifacts, and connect sessions, or narrow these storage-owned record types behind smaller package-local vocabularies at the consuming seams.

2. Missing best practice: agent auth-reference persistence still uses the legacy pair-of-columns schema.
   Evidence:
   - `001_ai.sql` stores `credential_id` and `provider_grant_id` on `ai_agents`.
   - `store_agents.go` still maps `agent.AuthReference` through those two columns.
   Why it matters:
   - P04’s typed-auth-reference cleanup cannot finish while the schema still encodes the old nullable-pair model.
   Refactor direction:
   - Introduce a cleaner auth-reference persistence shape and delete the implicit “exactly one of two IDs” schema contract.

3. Missing best practice: provider connect sessions are persisted through raw string fields rather than typed lifecycle vocabulary.
   Evidence:
   - `ProviderConnectSessionRecord` uses `Provider string` and `Status string`.
   - `service/provider_grant.go` constructs and interprets values like `"pending"` and `"completed"` manually.
   Why it matters:
   - This seam behaves like a domain lifecycle but has none of the domain package safety used elsewhere in the AI service.
   Refactor direction:
   - Either promote connect sessions into a typed domain/support package or shrink the persistence seam behind a narrower service-local repository contract with typed fields.

4. Refactor candidate: access-request persistence mixes generic upsert with workflow-specific CAS mutations in one interface.
   Evidence:
   - `AccessRequestStore` exposes `PutAccessRequest`, `ReviewAccessRequest`, and `RevokeAccessRequest`.
   - the sqlite implementation performs duplicated select-then-update CAS logic for review and revoke.
   Why it matters:
   - The contract is neither a pure repository nor a clearly command-oriented writer, which raises maintenance cost and duplicates state-transition persistence logic.
   Refactor direction:
   - Choose one model: either persist whole domain objects consistently or define explicit command-style mutation methods with a shared CAS helper.

5. Refactor candidate: `storage.Store` is a dead composite abstraction.
   Evidence:
   - `storage.go` defines the umbrella interface.
   - repository-wide search found no active use of `storage.Store`.
   Why it matters:
   - Dead aggregate interfaces make the contract layer look broader than it really is and encourage accidental omnibus dependencies later.
   Refactor direction:
   - Delete `storage.Store` unless a concrete consumer is added that genuinely benefits from it.

6. Missing best practice: the sqlite adapter duplicates validation, nil-store checks, and page assembly by hand across nearly every aggregate file.
   Evidence:
   - repeated `storage is not configured`, `page size must be greater than zero`, and scan-loop patterns across credentials, agents, provider grants, access requests, audit events, and debug turns.
   Why it matters:
   - Small policy changes to pagination or validation currently require scattershot edits across many files.
   Refactor direction:
   - Introduce a small set of sqlite-local helpers for repeated pagination/validation mechanics, or split the adapter into aggregate repositories with clearer shared infrastructure boundaries.

### Testability

7. Anti-pattern: coupled write workflows have no transaction seam to test or enforce atomicity.
   Evidence:
   - `ProviderGrantService.FinishConnect` persists a provider grant and then completes the connect session in separate store calls.
   - the storage layer exposes no transaction or unit-of-work boundary.
   Why it matters:
   - Multi-write workflows cannot express atomic intent, and tests cannot verify rollback/partial-write behavior at the repository seam.
   Refactor direction:
   - Add an explicit transaction runner or aggregate-specific atomic method for workflows that must mutate multiple persistence records together.

8. Missing best practice: pagination contracts are inconsistent and mostly stringly typed.
   Evidence:
   - most list methods use ascending `id` keyset tokens.
   - audit events use an autoincrement integer serialized as a string token.
   - campaign debug turns use a composite `started_at|id` token with descending sort.
   Why it matters:
   - Callers and contributors have to learn each list contract independently, and fake stores must reproduce those token rules by convention.
   Refactor direction:
   - Standardize pagination conventions where possible and document the exceptions explicitly where ordering genuinely differs.

9. Refactor candidate: tests still need white-box DB access for some assertions.
   Evidence:
   - `store_audit_events_test.go` uses `store.DB().QueryRowContext(...)` directly.
   - `Store.DB()` is exported on the concrete sqlite adapter.
   Why it matters:
   - This encourages tests to reach through the repository seam instead of asserting the published contract.
   Refactor direction:
   - Prefer contract-level tests through repository methods and remove `DB()` unless there is a durable runtime need for it beyond tests.

10. Missing best practice: migration coverage is thin and mostly incidental.
    Evidence:
    - `store_runtime_test.go` only checks that `Open("")` fails.
    - migration package files have no direct tests.
    - most migration safety comes indirectly from opening a fresh temp DB in aggregate tests.
    Why it matters:
    - Contributors changing schema or indexes do not have a focused test surface telling them which migration guarantees matter.
    Refactor direction:
    - Add migration-focused tests for schema bootstrap and key constraints/indexes, especially around auth references, unique labels, and debug-trace ordering.

### Contributor Clarity

11. Missing best practice: the docs overstate storage/domain alignment.
    Evidence:
    - `ai-service-architecture.md` says there are no separate storage record types.
    - the actual storage package still exports multiple record/filter/page types.
    Why it matters:
    - New contributors will start with the wrong mental model for changing artifacts, audit events, or provider connect sessions.
    Refactor direction:
    - Update docs after the contract cleanup so contributor guidance matches the actual storage surface.

12. Positive seam to preserve: sqlite behavior is already split by aggregate file and test file.
    Evidence:
    - separate `store_agents.go`, `store_credentials.go`, `store_provider_grants.go`, `store_access_requests.go`, and corresponding tests.
    Why it matters:
    - Contributors can localize most persistence edits once the contract vocabulary is cleaned up.
    Preservation note:
    - Keep per-aggregate files/tests even if shared helpers are introduced.

Target repository shape after refactor:

- domain aggregates use domain-owned store interfaces and domain-owned page/filter vocabulary
- support workflows such as artifacts, audit events, and connect sessions either gain real typed support packages or shrink behind narrower consumer-owned interfaces
- transaction-capable workflows get an explicit atomic boundary instead of ad hoc multi-call sequencing
- sqlite keeps one runtime root but exposes smaller repository implementations or helpers rather than one broad bag of repeated method patterns

Concrete refactor slices for a later implementation batch:

1. Delete the unused `storage.Store` umbrella interface.
2. Finish the `agent.AuthReference` storage cutover by replacing the dual-ID schema contract with one cleaner persistence representation.
3. Decide whether provider connect sessions deserve a typed support/domain package; if not, narrow the storage contract to a smaller lifecycle-specific seam.
4. Introduce a transaction runner or aggregate-specific atomic persistence method for workflows like provider-connect completion.
5. Standardize pagination helper patterns and document intentional exceptions such as newest-first debug traces.
6. Add migration-focused tests and reduce white-box DB assertions that bypass repository methods.

Tests to add, move, or delete in the refactor phase:

- Add direct migration bootstrap/constraint tests for `001_ai.sql` successors.
- Add contract tests for any new transaction runner or atomic store method.
- Keep aggregate-local sqlite tests, but reduce direct `store.DB()` assertions where repository methods can express the same contract.
- Update fake stores under `internal/test/mock/aifakes` when record DTOs are deleted or narrowed.

Docs to update in the refactor phase:

- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`
- `internal/services/ai/storage/doc.go`
- `internal/services/ai/storage/sqlite/doc.go`

## Context and Orientation

Use:

- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`
- `internal/services/ai/storage/doc.go`
- `internal/services/ai/storage/storage.go`
- `internal/services/ai/storage/sqlite/doc.go`

## Plan of Work

Inspect:

- interface grouping and size
- record/domain coupling
- pagination/filter consistency
- scan/helper duplication
- upsert/CAS semantics
- migration readability and contributor safety
- whether tests cover durable storage contracts

## Concrete Steps

1. Map each store interface to its actual callers.
2. Identify write-capable interfaces that should be split or renamed.
3. Review sqlite helper functions for reusable vs aggregate-specific logic.
4. Record breaking cleanup candidates that improve fakeability or clarity.

## Validation and Acceptance

- `go test ./internal/services/ai/storage/sqlite`
- `go test ./internal/services/ai/...`

Acceptance:

- target repository shape is explicit
- pagination/filter contract issues are recorded
- migration/process impacts are named concretely

## Idempotence and Recovery

- Prefer deleting misleading composite interfaces over preserving them for internal compatibility.

## Artifacts and Notes

- Store shapes coupling unrelated workflows that should be revisited:
  - `AccessRequestStore` mixing generic persistence with review/revoke CAS commands
  - storage-owned record DTOs for artifacts, audit events, and provider connect sessions
  - debug-turn pagination using a special composite token while the rest of the storage package defaults to simple ID keyset paging

## Interfaces and Dependencies

Track any proposed changes to:

- `storage.*Store` interfaces
- record/filter/page types
- sqlite helper and migration boundaries
- transaction or unit-of-work seams for multi-write workflows
