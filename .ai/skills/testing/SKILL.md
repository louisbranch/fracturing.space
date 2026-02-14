---
name: testing
description: Test-driven development workflow and coverage guardrails
user-invocable: true
---

# Testing Skill

Test-driven development and coverage guardrails for this project.

## TDD Workflow

- **Red**: Write one small test for a single behavior and verify it fails before implementation.
- **Green**: Implement the minimum code required to pass the test.
- **Refactor**: Improve structure and clarity while keeping tests green.

## Coverage Guardrails

- Treat coverage as a regression signal, not a goal.
- When adding or changing production code, run `make cover` and report the coverage impact.
- Add or update tests for new behavior; if a change is test-neutral (docs/refactor), call it out explicitly.
- Keep coverage from regressing versus the current baseline; CI enforces non-regression.
- If you introduce generated outputs, update `COVER_EXCLUDE_REGEX` in `Makefile` so coverage reflects hand-written code.

## Reporting

Include a short coverage note in your response, even if you could not run `make cover` locally.

Examples:

- "Coverage: ran `make cover`, total 82.4% (baseline 82.5%, -0.1%)."
- "Coverage: not run (reason), CI non-regression gate will validate."

## Testability

See [Testability Practices](../../../docs/project/testability.md) for dependency injection and constructor patterns.

## Verification

Project-wide verification commands live in `AGENTS.md`.
