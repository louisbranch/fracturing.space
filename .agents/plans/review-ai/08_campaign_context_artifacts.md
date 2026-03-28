# P08: Campaign Context, Instruction Loading, Memory/Reference Corpus, and Artifact Policy

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This document must be maintained in accordance with `PLANS.md`.

## Purpose / Big Picture

Review the campaign-context support packages to determine whether instruction loading, memory structure, reference lookup, and artifact management have clear ownership and maintainable contributor-facing boundaries.

Primary scope:

- `internal/services/ai/campaigncontext`
- `internal/services/ai/campaigncontext/instructionset`
- `internal/services/ai/campaigncontext/memorydoc`
- `internal/services/ai/campaigncontext/referencecorpus`

## Progress

- [x] (2026-03-23 03:59Z) Reviewed context support package boundaries, file-loading policy, and composition call paths from `app`, `service`, and `gametools`.
- [x] (2026-03-23 04:18Z) Recorded findings and proposed target ownership map.
- [x] (2026-03-23 04:20Z) Validation complete: `go test ./internal/services/ai/campaigncontext/...` passed.
- [x] (2026-03-23 04:21Z) Checked AI integration target note: `go test ./internal/test/integration -run 'TestAIGMCampaignContextReplay|TestAIGMCampaignContextLiveCapture'` returned `[no tests to run]`, so this pass cannot currently rely on that command as meaningful coverage.

## Surprises & Discoveries

- The clean ownership map described in the architecture docs is mostly real at the package level, but the composition edge is still strongly Daggerheart-shaped.
- `memorydoc` is not the problem area. It is intentionally narrow, deterministic, and already easier to reason about than most neighboring packages.
- Instruction fallback policy is split across two packages: `campaigncontext` still owns an inline default skills document while `app/server.go` separately decides how partial instruction loads degrade.
- `referencecorpus` is presented as a game-system reference package, but the current implementation is effectively a Daggerheart corpus with repo-local playbook side loading.
- The documented integration command for campaign-context validation is stale in the same way as P07: the named regex does not match actual tests.

## Decision Log

- Decision: Preserve `memorydoc` as a tiny structural helper package.
  Rationale: Its current API is cohesive, deterministic, and easy to test. Later refactors should avoid pulling memory section parsing back into larger orchestration or tool packages.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat instruction composition and degradation policy as a single ownership seam that should move toward `instructionset`, not remain split between `campaigncontext` and `app`.
  Rationale: The current design duplicates responsibility for “what happens when instruction files are missing” across packages, which is both harder to test and harder for contributors to discover.
  Date/Author: 2026-03-23 / Codex

- Decision: Treat `referencecorpus` as a shrink-and-specialize candidate unless the project commits to a real multi-system corpus abstraction.
  Rationale: The current package name sounds generic, but the implementation and configuration are Daggerheart-specific. Future refactors should either make the abstraction real or rename/split it so contributors do not infer nonexistent extensibility.
  Date/Author: 2026-03-23 / Codex

## Outcomes & Retrospective

Findings are grouped by review goal and classified as missing best practice, anti-pattern, or refactor candidate.

### Maintainability

1. Anti-pattern: instruction ownership is split across `campaigncontext`, `instructionset`, and `app`.
   Evidence:
   - The architecture doc says agent instructions are markdown files on disk, not Go string literals.
   - `campaigncontext/artifacts.go` still contains `defaultSkillsMarkdown()`.
   - `app/server.go` separately implements partial-load degradation in `loadPromptInstructions`.
   Why it matters:
   - Contributors cannot tell where the canonical instruction policy lives: on disk, in the loader, or in hardcoded fallback strings.
   Refactor direction:
   - Move instruction composition plus degradation policy behind one `instructionset`-owned bundle loader.
   - Make artifact seeding consume resolved skills content rather than carrying its own inline fallback document.

2. Missing best practice: the system-extension path is still Daggerheart-shaped at the composition boundary.
   Evidence:
   - `campaigncontext.DaggerheartSystem` is the default system constant in the root package.
   - `app/server.go` hard-codes `LoadSkills(campaigncontext.DaggerheartSystem)`.
   - `referencecorpus/search.go` only accepts one supported system.
   - startup config currently mounts one `DaggerheartReferenceRoot`.
   Why it matters:
   - The docs describe adding future systems, but the code still assumes one system across instructions, references, and startup wiring.
   Refactor direction:
   - Either make system selection explicit in the composition root and supporting packages, or narrow the package names/docs so they accurately describe a Daggerheart-only implementation.

3. Refactor candidate: the root `campaigncontext` package still exposes storage-shaped records as its working vocabulary.
   Evidence:
   - `Manager` store methods and return values are `storage.CampaignArtifactRecord`.
   Why it matters:
   - The package that is supposed to own artifact policy still leaks storage-edge shapes into callers and tests.
   Refactor direction:
   - Introduce a small `campaigncontext.Artifact` type or similar package-local vocabulary, then map storage records at the adapter boundary.

4. Missing best practice: the reference corpus reads from both a configured root and repo-local sidecar docs via runtime path discovery.
   Evidence:
   - `referencecorpus.Corpus` loads `index.json` from the configured root.
   - `repo_playbooks.go` also discovers `docs/reference/daggerheart-playbooks` from the repo checkout using `runtime.Caller`.
   Why it matters:
   - The corpus content model is no longer obvious from configuration alone, and tests depend on repository layout in addition to fixture roots.
   Refactor direction:
   - Fold repo playbooks into an explicit configured corpus source, or move that material into a separate Daggerheart-specific provider that the composition root assembles intentionally.

### Testability

5. Missing best practice: instruction fallback behavior is tested indirectly through `app/server.go` instead of through one package-local contract.
   Evidence:
   - `instructionset` tests cover file loading and composition.
   - partial-load degradation is asserted in `app/server_test.go`, not in an instruction-bundle package.
   Why it matters:
   - The most important behavior for missing instruction files spans package boundaries and is harder to change safely.
   Refactor direction:
   - Add package-local tests around a richer `instructionset` bundle/resolve API and delete the split fallback policy.

6. Refactor candidate: repo playbook manifest tests duplicate tool names manually.
   Evidence:
   - `referencecorpus/repo_playbooks_test.go` maintains a local `declaredToolNames()` list.
   Why it matters:
   - The test guards a useful contract, but it is brittle and drifts when tool ownership changes.
   Refactor direction:
   - Re-home this validation nearer to the actual Daggerheart tool-family registry once P07’s catalog split happens, or derive the tool list from a shared Daggerheart-owned manifest.

7. Refactor candidate: the documented campaign-context integration command is currently ineffective.
   Evidence:
   - `go test ./internal/test/integration -run 'TestAIGMCampaignContextReplay|TestAIGMCampaignContextLiveCapture'` returned `[no tests to run]`.
   Why it matters:
   - This pass does not currently have a reliable integration-level command for its advertised validation surface.
   Refactor direction:
   - Update the verification guidance to match real test names, or add umbrella tests that intentionally match the documented command.

### Contributor Clarity

8. Missing best practice: the docs promise a clearer system-extension path than the current code actually provides.
   Evidence:
   - `campaign-ai-agent-system.md` describes adding a system through instruction files and context sources.
   - the implementation still requires Daggerheart-specific startup and reference-corpus edits outside those system-local assets.
   Why it matters:
   - A new contributor could follow the docs and still miss required changes in generic startup/config code.
   Refactor direction:
   - Update the architecture/contributor docs after the package cleanup so the documented extension path matches the actual code path.

9. Positive seam to preserve: artifact path policy is explicit and well-tested.
   Evidence:
   - `NormalizeArtifactPath` and writable-path rules are localized in `campaigncontext/artifacts.go`.
   - package tests cover allowed paths, read-only behavior, and normalization.
   Why it matters:
   - Contributors can find artifact-path rules in one place instead of reverse-engineering them from handlers or tools.
   Preservation note:
   - Keep this path-policy seam centralized even if artifact data types change.

10. Positive seam to preserve: `memorydoc` is small, stable, and contributor-friendly.
    Evidence:
    - section parsing/editing is isolated in one file with focused tests.
    Why it matters:
    - This is a good model for AI-support packages: narrow responsibility, no runtime wiring, and direct tests around durable behavior.
    Preservation note:
    - Avoid expanding `memorydoc` into a catch-all campaign document package.

Target ownership map after refactor:

- `campaigncontext` owns artifact naming/path policy and package-local artifact vocabulary.
- `instructionset` owns instruction discovery, composition order, and missing-file degradation policy.
- `memorydoc` remains a tiny structural helper for `memory.md`.
- a system-specific reference provider owns Daggerheart repo playbooks and any future Daggerheart-only corpus rules.
- the composition root assembles systems and content sources explicitly instead of hard-coding Daggerheart assumptions across multiple packages.

Concrete refactor slices for a later implementation batch:

1. Move instruction resolution and fallback policy into a richer `instructionset` API, then delete `defaultSkillsMarkdown()` and the split partial-load logic in `app/server.go`.
2. Introduce a package-local artifact type for `campaigncontext` and move storage-record mapping to adapters.
3. Decide whether `referencecorpus` is truly multi-system. If not, split or rename it around Daggerheart-specific ownership; if yes, remove the single-system constant and repo-layout assumptions.
4. Make startup/system registration explicit so adding a new system touches one composition root instead of multiple Daggerheart-coded seams.
5. Repair the campaign-context verification command so it matches real integration coverage.

Tests to add, move, or delete in the refactor phase:

- Add package-local tests for instruction bundle resolution and partial-file degradation.
- Keep artifact path-policy tests as package-local contract tests.
- Move repo playbook/tool-manifest validation next to the Daggerheart-owned tool family after the P07 catalog split.
- Update or replace the stale integration `-run` pattern with one that matches real campaign-context tests.

Docs to update in the refactor phase:

- `docs/architecture/platform/campaign-ai-orchestration.md`
- `docs/architecture/platform/campaign-ai-agent-system.md`
- `docs/reference/ai-service-contributor-map.md`
- package comments in `campaigncontext`, `instructionset`, and `referencecorpus` if package names or ownership narrow

## Context and Orientation

Use:

- `docs/architecture/platform/campaign-ai-orchestration.md`
- `docs/architecture/platform/campaign-ai-agent-system.md`
- `docs/reference/ai-service-contributor-map.md`

## Plan of Work

Inspect:

- ownership boundaries between artifacts, instructions, memory, and references
- file-loading behavior and fallback policy
- search/read helper cohesion
- artifact-path vocabulary and safety
- contributor discoverability for editing AI behavior files

## Concrete Steps

1. Read package comments and loaders/managers first.
2. Compare documented ownership to actual call paths from `app` and `orchestration`.
3. Record any package that is serving as a catch-all.
4. Propose clearer package seams and doc updates.

## Validation and Acceptance

- `go test ./internal/services/ai/campaigncontext/...`
- `go test ./internal/test/integration -run 'TestAIGMCampaignContextReplay|TestAIGMCampaignContextLiveCapture'`

Acceptance:

- ownership map for artifacts/instructions/memory/reference is explicit
- fallback/loading policy is understandable
- contributor docs changes are recorded where needed

## Idempotence and Recovery

- If context data is discovered to be orchestration-owned instead of campaigncontext-owned, update both this file and P06.

## Artifacts and Notes

- Durable doc decisions to promote later:
  - whether campaign-context support is truly multi-system or still Daggerheart-first
  - the canonical owner of instruction fallback/degradation behavior
  - the real contributor path for adding a new system’s instructions and reference corpus

## Interfaces and Dependencies

Track any proposed changes to:

- artifact manager/store seams
- instruction loader contracts
- reference corpus search/read contracts
- composition-root system registration and instruction-resolution contracts
