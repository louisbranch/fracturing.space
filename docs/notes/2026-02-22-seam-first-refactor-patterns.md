---
title: "2026-02 Refactor: Seam-First Boundaries"
parent: "Notes"
nav_order: 2
---

# 2026-02 Refactor: Seam-First Boundaries

## Context

This refactor wave targeted clarity and testability by moving orchestration and helper
logic out of handler-shaped files into owned, small modules while preserving behavior.

- No migrations were introduced.
- No schema changes were made.
- No direct projection writes were introduced; write flows remain event-driven.

## Repository-wide patterns established

### 1) Keep gRPC/MCP handlers thin

- Handler methods should only:
  - validate request presence when needed, and
  - delegate to an extracted orchestrator (`<Domain>Application`, `contentApplication`, etc.).
- The extracted type should own orchestration sequence, validation, and command/event intent.
- This keeps transport-specific wiring and domain behavior separate and reviewable.

### 2) Group helper behavior by concern

- Shared helper logic should live near the owner of orchestration behavior.
- For outcome flows, helper behavior moved to `outcome_helpers.go` to reduce mixed
  responsibilities in `actions_outcomes.go`.
- For content endpoints, orchestration moved to `content_application.go` while endpoint
  mapping remained in `content_service.go`.

### 3) Introduce seams at integration points

- For configurable behavior (auth, rate limiting, TLS, optional policy adapters):
  prefer explicit constructor-injected interfaces and optional nil-safe defaults.
- This keeps production behavior unchanged and allows focused test seams without
  runtime coupling.

### 4) Split by action family

- Campaign/character/session-heavy files were decomposed by cluster:
  content, conditions, countdowns, damage, outcomes, recovery, adversary, and session flows.
- Small delegation wrappers stay in API surface files; implementation and helpers move to
  dedicated files.

### 5) Split test concerns at the file level

- Large test files were stabilized by moving focused suites into domain-aligned companion files:
  - MCP transport/runtime behavior moved out of `server_test.go` into `server_runtime_test.go`.
  - MCP campaign tool/resource behavior moved into `server_campaign_test.go`.
  - MCP test fixtures/mocks moved into `server_test_fixtures_test.go`.
- This keeps each suite narrowly scoped and lowers the cognitive overhead for future test additions.

## Why this improves testability

- Smaller orchestration units are easier to isolate.
- Helper extraction avoids long handler files that mix protocol mapping, event lookup,
  command dispatch, and status formatting.
- Focused test files make behavior failures easier to localize to a subsystem.
- New seams make negative-path and policy-path tests possible without building full stacks.

## Postconditions to keep in future work

- Do not add new business logic to transport/delegation wrappers.
- Keep optional dependencies validated at constructor boundaries when possible.
- If extraction reaches behavior changes, follow the project TDD sequence and preserve
  coverage as required in `.agents/AGENTS.md` and `.agents/plans/*`.
