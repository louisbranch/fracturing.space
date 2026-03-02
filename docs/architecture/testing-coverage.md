---
title: "Testing policy"
parent: "Architecture"
nav_order: 10
status: canonical
owner: engineering
last_reviewed: "2026-03-01"
---

# Test-Driven Development Policy

## Intent

We follow test-driven development (TDD) for behavior changes. Coverage is a regression guardrail, not a fixed target. Until we re-establish a stable baseline, the priority is to avoid decreases when adding or changing production behavior.

## Expectations

- **Invariant**: For behavior changes, follow TDD end-to-end (test first, then minimal implementation, then refactor). Exceptions are limited to non-behavioral changes (docs-only or refactors with no behavior change), which must be explicitly called out.
- Start with a failing test that captures a single behavior, then implement the minimum code, then refactor safely.
- When you add or change production code, add or update tests for the behavior.
- If a change is test-neutral (docs-only, refactor with no behavior change), call it out explicitly.
- Run `make cover` and report the coverage impact with your changes.
- If `make cover` reports deleted files, retry with a fresh Go cache (for example `GOCACHE="$(mktemp -d)" make cover`).
- For core game-domain behavior changes, also run `make cover-critical-domain` to measure cross-package exercise with `-coverpkg`.

## CI non-regression gate

- Pull requests compare coverage to the baseline from the latest `main` build, stored in `coverage-baseline.txt` on the `badges` branch.
- PRs fail if coverage drops more than 0.1% below the baseline.
- Baseline is recorded from `make cover` on `main` and stored in `coverage-baseline.txt` on the `badges` branch.
- Pull requests also compare critical-domain `-coverpkg` coverage against `coverage-critical-domain-baseline.txt` on `badges`.
- PRs fail if critical-domain coverage drops more than 0.1% below that baseline.
- Pull requests also enforce package-level floors for critical paths.
- Package floors use `docs/reference/coverage-floors.json` as seed policy and ratchet upward from `coverage-package-floors.json` on the `badges` branch.
- PRs fail when a critical package drops more than `allow_drop` from its ratcheted floor.
- Main pushes update ratcheted package floors on `badges` when coverage improves.
- Core runtime game-domain floors currently include:
  - `action`, `aggregate`, `authz`, `bridge`, `bridge/daggerheart`, `bridge/daggerheart/domain`,
  - `bridge/daggerheart/profile`, `bridge/daggerheart/internal/mechanics`, `bridge/daggerheart/internal/reducer`,
  - `bridge/manifest`, `campaign`, `character`, `checkpoint`, `command`, `engine`,
  - `event`, `fork`, `invite`, `journal`, `module`, `participant`, `readiness`,
  - `replay`, `session`,
  - plus shared `joingrant`.
- Web architecture floors also include critical transport seams:
  - `web/app`, `web/composition`, `web/modules`,
  - `web/platform/httpx`, `web/platform/requestmeta`, `web/platform/weberror`,
  - `web/platform/modulehandler`, `web/platform/publichandler`, `web/platform/pagerender`.

## Critical domain coverpkg lens

- `make cover-critical-domain` instruments curated critical domain packages via `-coverpkg`.
- The resulting profile (`coverage-critical-domain.out`) captures cross-package execution that per-package coverage can miss.
- CI runs this target on all test jobs and uploads both `.out` and `.func` artifacts for inspection.
- Main pushes store the critical-domain baseline in `coverage-critical-domain-baseline.txt` on `badges`.

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
