# Rename participant `display_name` to `name` across APIs, domain, and DB

This ExecPlan is a living document and is governed by `PLANS.md`.

This is a breaking change with no backward compatibility.

## Purpose / Big Picture

- Replace participant identity field usage from `display_name` to `name` across gRPC, MCP, domain, storage projections, chat, and admin surfaces.
- Remove support for legacy `display_name` in new/updated contracts and handlers.
- No DB migration is required.
- Preserve current physical DB shape while mapping the existing `display_name` column into the new `name` domain/API fields.
- Follow the `schema` skill:
  - no `ALTER TABLE`/column rename in this change.
  - protobuf field ordering is not required to stay backward compatible.
  - regenerate with `make proto`.

## Impact Analysis (Break Change)

- Client/API impact: all consumers of participant fields must switch to `name` (proto field, JSON payloads, and generated bindings) before deployment.
- Domain/model impact: validation/rejection labels/messages that use "display name" wording should be reviewed to avoid mixed nomenclature.
- Storage impact: existing SQLite projection rows remain in `display_name`; reads/writes are remapped in SQL/query structs to keep persistence stable without migration.
- Event/projection impact: if historical event payloads rely on old keys, they must be regenerated/frozen for replay compatibility or accepted as part of the breaking scope.

Out of scope:
- compatibility aliases (`display_name`) in public contracts.
- introducing new data model semantics beyond rename.

## Progress

- [x] (2026-02-16) Baseline and impacted-component inventory created.
- [x] (2026-02-16) Red: add/update targeted tests for `name` contract boundaries before implementation.
- [x] (2026-02-16) Green: implement contract/domain/storage/API changes to `name`.
- [x] (2026-02-16) Green: align SQLite query/model mapping to existing `display_name` DB column.
- [x] (2026-02-16) Green: complete event payload + replay compatibility updates.
- [x] (2026-02-16) Green: regenerate protobufs and generated code.
- [x] (2026-02-16) Green: update scenario/seed tool call sites and error wording to `name` payload semantics.
- [x] (2026-02-16) Green: validate no DB migration by confirming no participant-relevant `ALTER TABLE` changes in `internal/services/game/storage/sqlite` and no new participant migration files.
- [x] (2026-02-16) Refactor and cleanup: remove stale references and docs drift.
- [ ] (2026-02-16) Close with outcomes + recovery notes.

## Surprises & Discoveries

- Participant `display_name` appears across multiple layers and is not a single-service/API rename.
- Projections and event replay paths also carry the old key and require migration-free compatibility handling.
- There are static/client payload touchpoints (`campaign-chat.js`, templates, admin labels) that do not surface through service/domain code paths.

## Decision Log

- Decision: hard-break rename to `name` with no compatibility fallback.
  - Date/Author: 2026-02-16 / pending
- Decision: no DB migration in this change.
  - Date/Author: 2026-02-16 / pending
- Decision: use SQL model-level mapping (`display_name` column → `Name` field) instead of DDL.
  - Date/Author: 2026-02-16 / pending
- Decision: keep TDD discipline for behavioral changes where practical.
  - Date/Author: 2026-02-16 / pending

## Scope Boundaries

- In-scope:
  - participant domain fields, API DTOs/messages, projection/application logic, chat/admin/web views, and sqlite participant query mapping.
- Explicitly out-of-scope:
  - Creator display names (`creator_display_name`) and auth/user profile `DisplayName` fields, which keep existing naming.
  - DB schema migration DDL, column renames, and runtime column swaps.

## Plan of Work

### Phase 1 — Contract + protocol
- Update `api/proto/game/v1/participant.proto`:
  - field `display_name` → `name` for all participant request/response messages.
  - do not preserve proto compatibility.
- Run `make proto`.
- Regenerate and update code references for `GetName`/`Name`.

### Phase 2 — Domain/event updates
- Update participant payload/state/fold/decider/event payload types and keys:
  - `internal/services/game/domain/participant/*`
  - `internal/services/game/domain/campaign/event/*`
- Update projection/applier logic where participant names are derived from event payloads.

### Phase 3 — Storage compatibility (no migration)
- Keep DB column `display_name` unchanged.
- Update:
  - `internal/services/game/storage/storage.go`
  - `internal/services/game/storage/sqlite/queries/participants.sql`
  - `internal/services/game/storage/sqlite/db/models.go`
  - `internal/services/game/storage/sqlite/db/participants.sql.go`
- Ensure SQL reads/writes are targeting `display_name` and mapping to `Name` in application structs.

### Phase 4 — Service/API consumers
- Update game API handlers/mappers:
  - `internal/services/game/api/grpc/game/*`
- Update MCP domain/service contract mapping:
  - `internal/services/mcp/domain/campaign.go`
- Update chat/admin/web consumers and UI data keys using participant labels.

### Phase 5 — Validation
- Add/fix tests before each Green phase (AGENTS/Go TDD guidance).
- Execute:
  - `make test`
  - `make integration`
  - `make proto`
- Verify:
  - no stale `display_name` in behavior-critical paths.
  - event replay still works with unchanged historical rows.
  - no migration files changed.
  - no `ALTER TABLE` appears in the diff.

Status note: `make proto` completed in this pass. Full `make test` completed before the last integration-only patches. `make integration` was restarted after some files were updated and needs one final re-run to confirm green.

## Artifacts

- `api/gen/go/game/v1/participant.pb.go` (must be regenerated)
- `internal/services/game/storage/sqlite/db/participants.sql.go` (manual or regen-safe updates)
- Updated plan entries in `.ai/plans/rename-participant-display-name-to-name.md` as tasks progress.
