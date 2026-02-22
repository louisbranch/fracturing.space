# Game Audit Boundary Realignment (Clean-Slate Refactor)

## Purpose / Big Picture

Move durable audit telemetry out of platform infrastructure and make it explicit that audit records are game-service-owned.
Keep OpenTelemetry (`internal/platform/otel`) as distributed tracing and keep audit persistence/semantics inside game service packages.
No live data exists, so all identifier-level naming adjustments can be applied directly without migration.

## Progress

- [x] 2026-02-22 00:00Z Create execution plan for game-specific audit refactor and naming cleanup.
- [x] 2026-02-22 00:00Z Rename audit storage API from `AppendTelemetryEvent` to `AppendAuditEvent` across game storage + SQL layer.
- [x] 2026-02-22 00:00Z Move game audit emitter implementation from `internal/platform/audit` to `internal/services/game/observability/audit`.
- [x] 2026-02-22 00:00Z Update game call sites to import and use game-local audit package.
- [x] 2026-02-22 00:00Z Replace platform audit placeholders with game-local `events` and `metrics` package docs and event name constants.
- [x] 2026-02-22 00:00Z Remove empty or obsolete `internal/platform/audit/*` placeholders.
- [x] 2026-02-22 00:00Z Run `go test ./...`, `make integration`, `make cover`, and `make proto`; report results.
- [x] 2026-02-22 00:00Z Preserve refactor rationale in docs and keep PR note-ready artifacts in scope docs.

## Surprises & Discoveries

- `internal/platform/audit` is currently imported only by game gRPC layers (`authorization` and `interceptors`), so migration surface is contained.
- No schema rename is required for `audit_events`; table and migration are already aligned to audit semantics despite method-level telemetry naming.
- Stored `internal/platform/audit/events` and `internal/platform/audit/metrics` are placeholder packages only and currently unused.

## Decision Log

- Decision: keep event payload names like `telemetry.authz.decision` and `telemetry.grpc.*` for compatibility in existing dashboards and tests.
  - Rationale: these values are externally visible event labels already referenced by ops docs and would require cross-team coordination to rename safely.
  - Status: adopted.
- Decision: keep compatibility of `AppendAuditEvent` within the current storage API instead of introducing an adapter layer.
  - Rationale: no platform sharing currently uses this API and direct rename reduces long-term ambiguity.
  - Status: adopted.
- Decision: no data migration needed.
  - Rationale: branch assumes no live data and clean-slate environment; no backfill/rewrite needed.
  - Status: adopted.

## Outcomes & Retrospective

- Expected outcome: one audit package under game service owns all durable audit writes while OTEL remains separate and unchanged.
- Expected outcome: all remaining references use `AppendAuditEvent` and "audit" terminology in API names and messages.
- Expected outcome: platform no longer hosts empty/ambiguous audit packages.
- Retrospective section will be updated after code freeze + verification.

## Context and Orientation

- Relevant files:
  - `internal/services/game/storage/storage.go`
  - `internal/services/game/storage/sqlite/store_telemetry.go`
  - `internal/services/game/storage/sqlite/queries/telemetry.sql`
  - `internal/services/game/storage/sqlite/db/telemetry.sql.go`
  - `internal/services/game/api/grpc/interceptors/telemetry.go`
  - `internal/services/game/api/grpc/interceptors/telemetry_test.go`
  - `internal/services/game/api/grpc/game/authorization.go`
  - `internal/services/game/api/grpc/game/authorization_test.go`
  - `internal/platform/audit/*` (to remove)

- Test seam review:
  - `fakeAuditStore` in interceptor tests can assert method call shape after rename.
  - `authzAuditStore` in authorization tests already models the same `AuditEventStore` behavior.

## Plan of Work

- Use a narrow, behavior-preserving refactor:
  - rename API method names and update tests to verify all expectations.
  - relocate emitter package into game service and update imports.
  - delete orphaned placeholder packages in platform.
  - re-run verification commands from AGENTS.

## Concrete Steps

1. Update storage abstraction and implementation to `AppendAuditEvent`, plus query naming.
2. Move/duplicate emitter package files from `internal/platform/audit` to `internal/services/game/observability/audit`.
3. Introduce local `events` and `metrics` docs (and optional constants) under game observability.
4. Update game auth and interceptor code/tests to use new package path and method names.
5. Remove `internal/platform/audit` after updates compile.
6. Run validation:
   - `go test ./...`
   - `make integration`
   - `make cover`

## Validation and Acceptance

- `go test ./...` passes.
- `make integration` passes.
- `make cover` passes.
- `make proto` succeeds.
- No references remain to `internal/platform/audit` and no remaining `AppendTelemetryEvent` identifiers in game packages.
- No data migration required (clean-slate environment confirmed).
- `internal/services/game/observability/audit/events` and `internal/services/game/observability/audit/metrics` are now the canonical local game audit event/metric extension points.

## Idempotence and Recovery

- If a step fails, roll back with `git restore` of modified files and re-run with a smaller incremental subset.
- Since no production data migration is involved, recovery is limited to schema-neutral API rename or package relocation rollback via git.

## Artifacts and Notes

- Commit this work as one behavioral refactor commit plus optional cleanup commit for docs/notes if needed.
- Include PR notes explaining no live-data impact.

## Interfaces and Dependencies

- `storage.AuditEventStore` remains the boundary between auditing emitters and persistence.
- `interceptors.AuditInterceptor` and `authorization.emitAuthzDecisionTelemetry` remain the runtime emit points for game-service audit writes.
- `internal/platform/otel` remains untouched for tracing and span IDs used inside audit event payload enrichment.
