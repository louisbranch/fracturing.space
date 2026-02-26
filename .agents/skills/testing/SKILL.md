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

- When production behavior changes, run `make cover` and report notable impact.
- If coverage drops, explain whether risk changed and add targeted tests when needed.
- If you introduce generated outputs, update `COVER_EXCLUDE_REGEX` in `Makefile` so coverage reflects hand-written code.

## Verification

- Run the project verification commands in `AGENTS.md` after code changes.
- If a command cannot run locally, report why and what risk remains.

## Testability

See [Testability Practices](../../../docs/project/testability.md) for constructor, dependency injection, and fake patterns.
