---
title: "Testing policy"
parent: "Policy and quality"
nav_order: 2
status: canonical
owner: engineering
last_reviewed: "2026-03-02"
---

# Testing policy

Canonical testing and coverage policy for production behavior changes.

## Intent

- Protect durable behavior, invariants, and contracts.
- Keep tests at the correct seam (unit vs integration vs scenario).
- Treat coverage as a non-regression guardrail, not a vanity target.

## Required workflow for behavior changes

1. Start with a failing test that captures the intended behavior.
2. Implement the minimum change to pass.
3. Refactor while keeping tests green.

Expected verification commands:

- `make test`
- `make integration`
- `make cover`

For game-domain behavior changes, also run:

- `make cover-critical-domain`

If a change is intentionally test-neutral (docs-only or no-behavior refactor),
call that out explicitly in the PR.

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
