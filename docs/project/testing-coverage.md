# Test-Driven Development Policy

## Intent

We follow test-driven development (TDD) for behavior changes. Coverage is a regression guardrail, not a fixed target. Until we re-establish a stable baseline, the priority is to avoid decreases when adding or changing production behavior.

## Expectations

- **Invariant**: For behavior changes, follow TDD end-to-end (test first, then minimal implementation, then refactor). Exceptions are limited to non-behavioral changes (docs-only or refactors with no behavior change), which must be explicitly called out.
- Start with a failing test that captures a single behavior, then implement the minimum code, then refactor safely.
- When you add or change production code, add or update tests for the behavior.
- If a change is test-neutral (docs-only, refactor with no behavior change), call it out explicitly.
- Run `make cover` and report the coverage impact with your changes.

## CI non-regression gate

- Pull requests compare coverage to the baseline from the latest `main` build, stored in `coverage-baseline.txt` on the `badges` branch.
- PRs fail if coverage drops more than 0.1% below the baseline.
- Baseline is recorded from `make cover` on `main` and stored in `coverage-baseline.txt` on the `badges` branch.

## Generated code exclusions

Coverage excludes generated sources so we measure hand-written code. The exclusion list lives in `Makefile` as `COVER_EXCLUDE_REGEX`.

If you introduce new generated outputs, update the regex to exclude them. Examples include:

- `api/gen/` (protobuf output)
- `internal/services/*/storage/sqlite/db/` (sqlc output)
- `*_templ.go` or `internal/services/admin/templates/` (templ output)

## Structuring code for testability

See [Testability Practices](testability.md) for dependency injection and constructor patterns that keep new code testable from the start.

## Raising the bar

Once the baseline is consistently stable, we can add a fixed target threshold on top of the non-regression gate.
