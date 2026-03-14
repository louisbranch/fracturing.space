---
title: "Testing policy"
parent: "Policy and quality"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-10"
---

# Testing policy

Canonical testing and coverage policy for production behavior changes.

## Intent

- Protect durable behavior, invariants, and contracts.
- Keep tests at the correct seam (unit vs integration vs scenario).
- Treat coverage as a non-regression guardrail, not a vanity target.

## Required workflow for behavior changes

1. Start with a failing test when introducing or changing durable behavior.
2. If behavior is intentionally removed, delete stale tests instead of preserving the historical path.
3. Implement the minimum change to pass.
4. Refactor while keeping tests green.

Use the canonical [Verification commands](../../running/verification.md)
workflow:

- `make test` during normal implementation
- `make smoke` when runtime paths need quick feedback
- `make check` before push, PR open, or PR update

Focused diagnostics remain available via `make cover`, `make docs-check`, and
`make cover-critical-domain` when you need standalone coverage output separate
from `make check`.

If a change is intentionally test-neutral (docs-only or no-behavior refactor),
call that out explicitly in the PR.

## Assertion policy

- Prefer positive assertions against durable contracts, outputs, and state transitions.
- Use negative assertions only for explicit invariants such as security, privacy, protocol framing, or mutual exclusion.
- Every negative assertion must include an adjacent `Invariant:` rationale explaining why the absence matters.
- Tombstone tests that only memorialize removed behavior are discouraged. If no durable invariant remains, delete the stale test.

## CI non-regression gates

PR coverage compares against baselines published from `main` on `badges`:

- overall baseline: `coverage-baseline.txt`
- critical-domain baseline: `coverage-critical-domain-baseline.txt`
- critical package floors: ratcheted from `coverage-package-floors.json`

PRs fail when coverage drops beyond configured tolerance.

Seed floor policy is versioned in:

- `docs/reference/coverage-floors.json`

## Testability requirements

Code that cannot be tested cannot be safely evolved. New code should follow
these constructor and dependency rules:

- Accept dependencies at constructors; do not create hard dependencies inside logic paths.
- Define interfaces at consumption points.
- Use function injection for simple dependencies (clock/random/IO hooks).
- Keep production constructors thin; expose internal constructors for tests when needed.
- Keep fakes local to `*_test.go` and implement only required methods.

## What not to unit test

- Thin production wiring (`main`, server composition glue)
- Generated code (`api/gen`, `sqlc`, `templ` outputs)
- CLI flag plumbing when logic is already tested in invoked functions

Use integration/scenario tests for cross-package workflows and transport seams.

## Generated code coverage exclusions

Coverage excludes generated sources via `COVER_EXCLUDE_REGEX` in `Makefile`.
Update it when new generated outputs are introduced.

## Related policy docs

- [Event payload change policy](event-payload-change-policy.md)
- [Architecture foundations](../foundations/architecture.md)
