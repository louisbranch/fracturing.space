---
name: testing
description: Meaningful testing strategy and coverage guardrails
user-invocable: true
---

# Testing Skill

Testing guidance focused on durable behavior and maintainable feedback loops.

## Core Principles

- Tests protect durable contracts, invariants, and failure modes.
- Prefer tests that still matter after refactors, removals, and package moves.
- Coverage informs risk; it is not a target to game.

## Choose the Right Test Level

- Unit tests: deterministic domain logic, pure transformations, validation rules.
- Integration tests: seams between transport, domain, storage, and adapters.
- End-to-end tests: critical user/system paths only.
- During architecture-first refactors, favor seam/integration coverage around stable contracts before cutover.

## Test-First Guidance (Not Ceremony)

- Prefer test-first when it clarifies behavior or de-risks implementation.
- For large refactors or testability seam setup, it is acceptable to reshape code first, then add or adjust tests before declaring the change done.
- Do not create ceremonial failing tests for behavior intentionally removed.
- When behavior is removed, delete stale tests and replace them only with tests for the new intended contract.

## Durable Assertion Heuristics

- Start with a user-visible contract: what should render, enable, redirect, or block.
- Prefer positive assertions (`contains`, state transitions, status, links) over absence assertions.
- Use negative assertions only for explicit invariants (security/privacy/protocol/mutual exclusion).
- Every allowed negative assertion should include an adjacent `// Invariant: ...` rationale.

Example:

- Weak assertion: "response does not contain class `foo`."
- Strong assertion: "HTMX response returns fragment content and does not include a full HTML document wrapper (`Invariant:` protocol contract)."

## Coverage Guardrails

- Use `make cover` and `make cover-critical-domain` only when you need focused
  standalone coverage diagnostics outside the normal verification workflow.
- If coverage drops, explain whether risk changed and add targeted tests when needed.
- If you introduce generated outputs, update `COVER_EXCLUDE_REGEX` in `Makefile` so coverage reflects hand-written code.

## Verification

- Use the public verification surface from `docs/running/verification.md`:
  - `make test` during normal implementation
  - `make smoke` when runtime paths need quick feedback
  - `make check` before pushing or updating a PR
- Do not run `make cover*` in parallel with `make check`; `make check` already
  generates the shared coverage artifacts.
- When a verification command is running in the background, inspect
  `.tmp/test-status/` instead of repeatedly re-running or blindly polling the
  process.
- Use `.tmp/test-status/test/status.json` for `make test`.
- Use `.tmp/test-status/smoke/status.json` for overall `make smoke` stage
  progress.
- Use `.tmp/test-status/smoke/integration/status.json` and
  `.tmp/test-status/smoke/scenario/status.json` for lane-specific `make smoke`
  progress.
- Use `.tmp/test-status/check/status.json` for `make check` stage progress.
- If `make check` is in `check-runtime`, read
  `.tmp/test-status/check-runtime/scenario/status.json`.
- If `make check` is in `check-coverage`, read `.tmp/test-status/cover/status.json`
  and the nested shard status files under `.tmp/test-status/cover/`.
- Treat `state`, `current_stage`, `current_package`, `current_test`,
  `packages_completed`, `packages_running`, `updated_at_utc`, and
  `last_event_at_utc` as the primary fields for liveness and progress.
- If a command cannot run locally, report why and what risk remains.

## Removal Policy

- Do not preserve tests that only assert a removed feature, path, or section stays gone.
- If no durable invariant remains after a removal, delete the stale test.
- If a negative assertion must remain, add an adjacent `Invariant:` rationale.

## Testability

See `docs/architecture/policy/testing-policy.md` for constructor, dependency injection, and fake-oriented testability guidance.
