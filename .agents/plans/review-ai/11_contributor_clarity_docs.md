# P11: Contributor Clarity, Docs, Naming, Package Comments, and Reading Order

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the contributor-facing clarity of the AI service: naming, package comments, docs accuracy, reading order, and whether durable terms and architecture explanations match the actual implementation.

Primary scope:

- AI package comments and exported names
- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`
- `docs/reference/ai-service-lifecycle-terms.md`
- other AI-specific docs touched by review findings

## Progress

- [x] (2026-03-23 03:59Z) Reviewed canonical AI docs, package comments, naming, and contributor reading flow against the live package structure.
- [x] (2026-03-23 05:02Z) Recorded findings and proposed durable doc cleanup.
- [x] (2026-03-23 05:03Z) Validation complete: `go test ./internal/services/ai/...` passed.

## Surprises & Discoveries

- Most AI packages already have package comments; the bigger issue is not missing comments but comments and docs that overstate architectural cleanliness.
- The highest-friction contributor drift is in extension guidance: multiple docs still describe the old monolithic `gametools` path as the way to add tools or systems.
- The composition-root directory/package mismatch (`internal/services/ai/app` vs `package server`) is a small but recurring source of lookup friction.
- The lifecycle-terms doc is ahead of some code edges conceptually, but it reads more “target architecture” than “current state” in places like `AuthReference`.
- Secondary reference docs (`ai-tools.md`, `ai-resources.md`) have lower signal quality than the canonical architecture docs and currently amplify outdated structural assumptions.

## Decision Log

- Decision: Keep the contributor map as the primary reader-first routing document, but narrow it to stable entrypoints and remove temporary implementation details from its recommendations.
  Rationale: The file is already the best starting point for new contributors. The issue is that it currently blesses unstable or shrink-target surfaces like `transport_test_helpers_test.go`.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat reference docs that point directly at shrink-target implementation files as cleanup candidates unless they describe a stable contract.
  Rationale: Several AI reference docs currently steer contributors into monolithic implementation files that this review is already flagging for refactor.
  Date/Author: 2026-03-23 / Codex

- Decision: Prefer honest “current ownership and likely next seam” language over polished-but-inaccurate architecture claims.
  Rationale: P11 found that contributor confusion mostly comes from docs that sound cleaner than the current code really is.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

Findings are grouped by review goal and classified as missing best practice, anti-pattern, or refactor candidate.

### Maintainability

1. Anti-pattern: the architecture docs overstate cleanliness in areas the review has already identified as active debt.
   Evidence:
   - `docs/architecture/platform/ai-service-architecture.md` says there are no separate storage record types.
   - P09 found storage-owned record DTOs still exist for artifacts, audit events, and provider connect sessions.
   - the same doc’s “How to Add a Game System Tool” section still routes contributors through the monolithic `gametools` registry path.
   Why it matters:
   - Contributors start with a mental model that does not match the code they will immediately touch.
   Refactor direction:
   - Update the architecture doc to describe the current seams honestly and point to the intended post-refactor direction only where explicitly labeled.

2. Missing best practice: contributor-facing docs still normalize shrink-target files as primary extension surfaces.
   Evidence:
   - `docs/reference/ai-service-contributor-map.md` explicitly points test contributors to `transport_test_helpers_test.go`.
   - `docs/reference/ai-tools.md` and `docs/reference/ai-resources.md` point readers at `orchestration/gametools/tools.go`.
   - `docs/guides/adding-command-event-system.md` still tells system/tool contributors to register production AI tooling directly in the monolithic `gametools` path.
   Why it matters:
   - These docs direct new contributors into the exact surfaces P07 and P10 marked for decomposition.
   Refactor direction:
   - Rewrite contributor guidance around stable ownership boundaries and composition points, not around current monolith files.

3. Refactor candidate: package naming and directory naming are slightly misaligned at the composition root.
   Evidence:
   - contributors are told to read `internal/services/ai/app/`.
   - the actual package name in that directory is `server`.
   Why it matters:
   - This is small, but it adds avoidable confusion when navigating imports, package docs, and grep output.
   Refactor direction:
   - Either rename the package to `app` or update docs/package comments to call out the mismatch explicitly so readers do not have to infer it.

4. Refactor candidate: `referencecorpus` is documented too generically for the current implementation.
   Evidence:
   - package comment says “game-system reference corpus.”
   - P08 found the implementation is effectively Daggerheart-specific today.
   Why it matters:
   - Contributors may assume multi-system behavior and extension points that do not actually exist yet.
   Refactor direction:
   - Either specialize the docs/package comment around Daggerheart-first ownership or finish the multi-system abstraction before continuing to describe it generically.

### Testability

5. Missing best practice: contributor docs still recommend unstable test-entry surfaces.
   Evidence:
   - contributor map tells readers to add handler test setup in `transport_test_helpers_test.go`.
   - P10 found that file is a 718-line shrink target and effectively a custom local framework.
   Why it matters:
   - New contributors are being told to add more weight to the least maintainable test helper in the AI service.
   Refactor direction:
   - Update the docs to prefer handler-family-local helpers and service-level tests over expansion of the shared transport harness.

6. Missing best practice: verification guidance is partially stale and underspecified for AI integration tests.
   Evidence:
   - contributor map says `go test ./internal/test/integration -tags=integration -run 'TestAIGM|TestGameEndToEnd'`.
   - P07/P08/P10 found several AI-specific commands in plans were stale or tag-incomplete.
   - live-capture tests additionally require `liveai`.
   Why it matters:
   - Contributors can run a command that sounds correct but misses the intended AI coverage.
   Refactor direction:
   - Publish one explicit AI integration command set with tag requirements and examples for replay/direct-session vs live-capture runs.

### Contributor Clarity

7. Missing best practice: lifecycle vocabulary docs describe some cleanup goals as if they are already fully true across all layers.
   Evidence:
   - `docs/reference/ai-service-lifecycle-terms.md` says transport and storage should project typed `AuthReference` rather than reimplement exclusivity rules.
   - P04 and P09 found sqlite and edge layers still carry legacy projections.
   Why it matters:
   - The vocabulary doc is useful, but it currently blends current-state guidance with target-state architecture in a way that can mislead readers.
   Refactor direction:
   - Mark target-state guidance more explicitly where the codebase is still mid-cutover, or align the implementation before keeping the stronger wording.

8. Positive seam to preserve: package comments exist for nearly every AI package and are generally purposeful.
   Evidence:
   - the AI tree has package comments for domain, transport, orchestration, campaigncontext, storage, and support packages.
   Why it matters:
   - Contributors already have a better package-level reading experience here than in many codebases.
   Preservation note:
   - The next step is accuracy and calibration, not a wholesale rewrite.

9. Positive seam to preserve: the contributor map’s reading order is still fundamentally correct.
   Evidence:
   - starting from composition root, then transport, service, domain, orchestration, support packages, and storage matches how live behavior is wired.
   Why it matters:
   - The navigation skeleton is good; the problem is stale specifics, not overall structure.
   Preservation note:
   - Keep the reading order, but refresh the package-role descriptions and test guidance.

10. Refactor candidate: AI reference docs are unevenly trustworthy.
    Evidence:
    - `ai-tools.md` duplicates a `tools.go` reference and points readers at implementation files rather than stable contracts.
    - `ai-resources.md` is concise but also routes readers straight to monolithic implementation files.
    Why it matters:
    - Contributors cannot easily tell which AI docs are canonical architecture vs low-level implementation notes.
    Refactor direction:
    - Either upgrade these references into stable contract docs or slim them down and point readers back to the canonical architecture/contributor pages.

Minimum durable doc-update set after refactor:

- `docs/architecture/platform/ai-service-architecture.md`
- `docs/reference/ai-service-contributor-map.md`
- `docs/reference/ai-service-lifecycle-terms.md`
- `docs/reference/ai-tools.md`
- `docs/reference/ai-resources.md`
- `docs/guides/adding-command-event-system.md`
- package comments for `internal/services/ai/orchestration/gametools`, `internal/services/ai/storage`, and possibly `internal/services/ai/campaigncontext/referencecorpus`

Concrete refactor slices for a later implementation batch:

1. Remove or rewrite doc claims that still state “no separate storage record types.”
2. Rewrite “add a tool / add a system” guidance so it no longer hard-codes the monolithic `gametools` path as the long-term extension story.
3. Refresh the contributor map’s “Where to add tests” section to stop routing contributors into `transport_test_helpers_test.go`.
4. Clarify package/directory naming at the composition root.
5. Distinguish current-state terminology from target-state terminology in the lifecycle terms doc where cutovers are still incomplete.

Docs to update first if only a minimal cleanup lands:

- contributor map
- architecture overview
- AI tools/resources references

## Context and Orientation

Use:

- `docs/architecture/foundations/domain-language.md`
- `docs/reference/ai-service-contributor-map.md`
- `docs/reference/ai-service-lifecycle-terms.md`

## Plan of Work

Inspect:

- missing or weak package comments
- naming drift between docs and code
- stale contributor routing guidance
- whether reading order matches the live composition root
- whether durable vocabulary is being redefined informally in tests or helpers

## Concrete Steps

1. Compare docs claims to live package structure and wiring.
2. Inventory package comments and obvious documentation gaps.
3. Record stale terms, ambiguous names, and packages that are hard to place conceptually.
4. Produce a minimum durable doc-update set.

## Validation and Acceptance

- `go test ./internal/services/ai/...`

Acceptance:

- contributor reading path is explicit
- stale docs/naming are recorded
- package-comment and terminology changes are concrete enough to implement

## Idempotence and Recovery

- Durable docs belong in `docs/`; pass-local working notes stay here.

## Artifacts and Notes

- Lasting ownership/vocabulary decisions to promote:
  - whether `gametools` remains a generic extension seam or is explicitly repositioned as a shrinking implementation package
  - whether `referencecorpus` is documented as Daggerheart-first or truly multi-system
  - the canonical AI integration command set with required build tags

## Interfaces and Dependencies

Track any proposed changes to:

- exported names and package comments
- AI reference and architecture docs
- contributor routing and verification guidance
