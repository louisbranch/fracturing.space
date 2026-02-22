---
title: "2026-02 Newcomer Refactor Program"
parent: "Notes"
nav_order: 1
---

# 2026-02 Newcomer Refactor Program

## Status

Accepted and implemented incrementally during refactor workstream.

## Context

Onboarding friction for first-time contributors was concentrated in a small set
of multi-thousand-line files that mixed unrelated responsibilities.

High-friction hotspots were:

- `internal/services/game/storage/sqlite/store.go`
- `internal/services/game/api/grpc/systems/daggerheart/actions.go`
- `internal/services/admin/handler.go`
- `internal/services/mcp/domain/campaign.go`

## Decisions

1. Decompose hotspot files by bounded context and capability, using move-only
   slices first to preserve runtime behavior.
2. Enforce constructor-time dependency validation for Daggerheart services so
   misconfiguration fails fast at startup.
3. Introduce shared game test fakes under `internal/test/mock/gamefakes` and
   migrate low-risk duplicate test fakes to improve test readability.
4. Centralize Daggerheart command/event type identifiers behind typed constants
   to reduce string-literal drift in production call sites.

## Rationale

- Smaller files improve code navigation and reduce first-change risk.
- Explicit constructor validation removes repetitive nil-guard noise from
  handlers and makes bootstrap failures deterministic.
- Shared fakes lower test duplication and make error-path tests easier to add.
- Typed identifiers improve discoverability and reduce typo regressions.

## Consequences

- Refactor-only slices can cause noisy repository-level coverage fluctuations
  when using broad `-coverpkg=./...`; canonical coverage should be read from
  `go tool cover -func=coverage.out | tail -n 1`.
- Some identifier-centralization work remains outside Daggerheart handlers and
  should continue in future incremental slices.
- MCP campaign pagination TODO remains intentionally deferred as a separate,
  behavior-changing TDD task.
