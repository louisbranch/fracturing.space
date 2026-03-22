# Pass 10: Storage Interface Granularity

**Date:** 2026-03-22
**Scope:** `internal/services/game/storage/` contracts, `storage/sqlite/eventjournal/`, `storage/sqlite/coreprojection/`, `storage/sqlite/integrationoutbox/`, `storage/sqlite/projectionapplyoutbox/`, `storage/integrity/`, `storage/sqlite/migrations/`

## Summary

The storage layer is well-architected. The Reader/Store split is consistent and clean across all six contract files. Interfaces are defined at consumption points with purpose-scoped composites (`CampaignReadStores`, `SessionReadStores`, `SceneReadStores`) that avoid over-embedding. SQLite implementations uniformly use parameterized queries through sqlc-generated code and hand-written prepared statements. The event journal enforces append-only semantics with database triggers, provides robust chain integrity verification, and uses atomic transactions for event-plus-outbox writes. Migration files include both Up and Down sections. Test coverage is thorough across event lifecycle, outbox processing, integrity verification, and edge cases.

Findings are minor. The most notable items are: one raw SQL UPDATE in `BatchAppendEvents` bypassing the sqlc layer, repetitive event-row conversion boilerplate, and the `ListEventsPage` method building dynamic SQL that could benefit from more structured plan validation.

---

## Findings

### 1. Raw SQL UPDATE in BatchAppendEvents bypasses sqlc layer
**Category:** missing best practice
**File:** `internal/services/game/storage/sqlite/eventjournal/store_events_append.go:300-305`

`BatchAppendEvents` uses a raw `tx.ExecContext` call to advance the sequence counter:
```go
if _, err := tx.ExecContext(ctx,
    "UPDATE event_seq SET next_seq = ? WHERE campaign_id = ?",
    nextSeq, campaignID,
); err != nil {
```
All other sequence operations (`InitEventSeq`, `GetEventSeq`, `IncrementEventSeq`) go through the sqlc-generated `db.Queries` layer. This raw SQL diverges from the pattern used by `AppendEvent` (which calls `IncrementEventSeq` per-event). The raw UPDATE is correct and parameterized, but it creates a maintenance risk: if the `event_seq` table schema changes, this statement won't be caught by sqlc regeneration.

**Proposal:** Add a `SetEventSeq(ctx, campaignID, nextSeq)` query to the sqlc definitions and use it here, or document this intentional bypass. Alternatively, loop `IncrementEventSeq` isn't viable for batch, so a dedicated sqlc query is the right fix.

---

### 2. Repetitive event row conversion boilerplate
**Category:** contributor friction
**File:** `internal/services/game/storage/sqlite/eventjournal/store_events_convert.go:104-210`

Five nearly identical functions (`eventRowDataFromGetEventByHashRow`, `eventRowDataFromGetEventBySeqRow`, `eventRowDataFromListEventsRow`, `eventRowDataFromListEventsBySessionRow`, `eventRowDataFromEvent`) each copy 22 fields from slightly different sqlc-generated row types into the same intermediate `eventRowData` struct. This is a consequence of sqlc generating distinct row types per query.

This is not a bug, and the current approach is type-safe. However, adding a new event column requires updating all five functions plus the intermediate struct.

**Proposal:** Consider using a generic conversion helper or a single raw scan function if the field set is always identical. Alternatively, accept this as the cost of type safety from sqlc-generated code and document it as intentional.

---

### 3. ListEventsPage dynamic SQL construction is safe but could be more defensive
**Category:** missing best practice
**File:** `internal/services/game/storage/sqlite/eventjournal/store_events_query.go:170-176` and `list_events_page_plan.go:20-78`

The `ListEventsPage` method builds SQL by interpolating `whereClause`, `orderClause`, and `limitClause` from `listEventsPageSQLPlan` using `fmt.Sprintf`. All user-supplied values are properly parameterized via `?` placeholders -- there is no SQL injection risk. The `LIMIT` clause safely uses `fmt.Sprintf("LIMIT %d", req.PageSize+1)` with an integer.

However, the plan builder doesn't validate that `CursorDir` is one of `"fwd"` or `"bwd"`. An unexpected value silently falls through to the `else` branch (treating it as forward). This is not a correctness bug since unknown values produce valid SQL, but explicit validation would prevent silent misconfiguration.

**Proposal:** Add a validation guard at the top of `buildListEventsPageSQLPlan` that rejects unknown `CursorDir` values (when non-empty) with an error.

---

### 4. EventStore interface combines read and write without a Reader split
**Category:** anti-pattern (minor)
**File:** `internal/services/game/storage/contracts_events_audit_stats.go:12-30`

Unlike `CampaignStore`/`CampaignReader`, `SessionStore`/`SessionReader`, etc., the `EventStore` interface has no `EventReader` counterpart. It bundles read methods (`GetEventByHash`, `GetEventBySeq`, `ListEvents`, `ListEventsBySession`, `GetLatestEventSeq`, `ListEventsPage`) with the write method (`AppendEvent`) in a single interface. Consumers that only need to read events (replay engines, query handlers, admin tooling) must accept the full `EventStore` including write capability.

The project convention doc states "Define interfaces at consumption points," so this is a minor deviation. The event journal is intentionally append-only and single-implementation, which reduces the practical impact.

**Proposal:** Extract an `EventReader` interface for the six read-only methods. This would let query handlers and replay consumers accept a narrower contract. The `EventStore` would embed `EventReader` plus `AppendEvent`.

---

### 5. AuditEventStore is append-only with no Reader
**Category:** missing best practice (minor)
**File:** `internal/services/game/storage/contracts_events_audit_stats.go:105-107`

`AuditEventStore` only has `AppendAuditEvent`. There is no `AuditEventReader` for querying audit records. This is likely intentional at this stage (audit records may be read through direct SQL or observability tooling), but it means there's no contract-level seam for testing audit read paths.

**Proposal:** No action needed now. When audit read use cases emerge (admin dashboards, compliance exports), an `AuditEventReader` interface should be added.

---

### 6. SceneGate uses raw []byte for MetadataJSON/ResolutionJSON; SessionGate uses map[string]any
**Category:** contributor friction
**File:** `internal/services/game/storage/contracts_session_scene.go:202-218` (SceneGate) and `contracts_session_scene.go:55-72` (SessionGate)

`SessionGate.Metadata` and `SessionGate.Resolution` are typed as `map[string]any`, while `SceneGate.MetadataJSON` and `SceneGate.ResolutionJSON` are `[]byte`. These represent the same conceptual data (gate metadata and resolution) at different abstraction levels. The inconsistency means consumers of SceneGate must unmarshal themselves, while SessionGate consumers get pre-decoded maps.

This is a deliberate design choice (scene gates are newer and may be optimizing for pass-through), but the naming inconsistency (`Metadata` vs `MetadataJSON`) could confuse contributors.

**Proposal:** Align the representation: either both use `map[string]any` (decoded at storage boundary) or both use `[]byte` (decoded at consumer). Prefer `map[string]any` for consistency with SessionGate.

---

### 7. IntegrationOutboxStore has no Reader split
**Category:** anti-pattern (minor)
**File:** `internal/services/game/storage/contracts_integration_outbox.go:40-47`

`IntegrationOutboxStore` bundles read (`GetIntegrationOutboxEvent`) with write and lifecycle operations (`Enqueue`, `Lease`, `MarkSucceeded`, `MarkRetry`, `MarkDead`). For worker consumers that only need to lease and process, the full interface is appropriate. But for monitoring/inspection use cases, a narrower `IntegrationOutboxReader` would be cleaner.

**Proposal:** Low priority. The outbox is a worker-facing internal concern with a single implementation. Consider splitting only when monitoring consumers emerge.

---

### 8. Projection composite ProjectionStore embeds Store interfaces (not Readers)
**Category:** anti-pattern
**File:** `internal/services/game/storage/contracts_projection_state.go:118-132`

`ProjectionStore` embeds `CampaignReadStores`, `SessionReadStores`, and `SceneReadStores`, which themselves embed the full `*Store` interfaces (e.g., `CampaignStore`, `SessionStore`). This means `ProjectionStore` includes write methods (`Put`, `PutSession`, `PutScene`, etc.) that are only needed by projection handlers.

Read-only API consumers accepting `ProjectionStore` get write capabilities they shouldn't use. The `CampaignReadStores` composite name is misleading -- it actually embeds `CampaignStore` (which includes `CampaignReader` + `Put`), not just `CampaignReader`.

**Proposal:** Rename `CampaignReadStores` to `CampaignStores` (removing the "Read" qualifier that implies read-only), or create true read-only composites (`CampaignReadStores` embedding only `CampaignReader` + `ParticipantReader` + ...) and use those for API consumers. The current naming is the main friction point.

---

### 9. ProjectionApplyTxStore re-embeds ProjectionWatermarkStore alongside the three read-store composites
**Category:** correctness risk (low)
**File:** `internal/services/game/storage/contracts_projection_apply.go:74-80`

`ProjectionApplyTxStore` embeds `CampaignReadStores`, `SessionReadStores`, `SceneReadStores`, and `ProjectionWatermarkStore`. Since `ProjectionStore` also embeds these same interfaces plus `SnapshotStore` and `StatisticsStore`, there's no diamond-problem risk in Go (methods resolve unambiguously). However, the parallel composition makes it easy for a contributor to forget that `ProjectionApplyTxStore` intentionally excludes `SnapshotStore` and `StatisticsStore` from the transaction scope.

**Proposal:** Add a doc comment to `ProjectionApplyTxStore` explicitly noting the exclusions and why. The current comment ("Core projection stores only") is helpful but could be stronger.

---

### 10. Event journal sequence allocation uses three separate queries
**Category:** missing best practice
**File:** `internal/services/game/storage/sqlite/eventjournal/store_events_append.go:50-60`

`AppendEvent` calls `InitEventSeq`, `GetEventSeq`, and `IncrementEventSeq` as three separate SQL operations within the same transaction. This is safe because they run in a serialized transaction, but an `INSERT ... RETURNING` or `UPDATE ... RETURNING` pattern could reduce to a single round-trip.

SQLite supports `RETURNING` since 3.35.0. The current approach is explicit and easy to follow.

**Proposal:** Low priority. The three-query approach is clear and correct. Consider consolidating only if event append latency becomes a concern.

---

### 11. Event journal append-only triggers protect immutability at the database level
**Category:** (positive finding)
**File:** `internal/services/game/storage/sqlite/migrations/events/001_events.sql:64-74`

The `events_no_update` and `events_no_delete` triggers in the event schema enforce append-only semantics at the SQL level, preventing even direct SQL modifications. This is an excellent defense-in-depth measure that complements the application-level integrity checks.

---

### 12. Projection-apply outbox FK references events table for referential integrity
**Category:** (positive finding)
**File:** `internal/services/game/storage/sqlite/migrations/events/003_projection_apply_outbox.sql:13`

The `projection_apply_outbox` table uses `FOREIGN KEY (campaign_id, seq) REFERENCES events(campaign_id, seq) ON DELETE CASCADE`, ensuring orphan outbox rows cannot exist without a corresponding event. This is tested explicitly in `TestProjectionApplyOutboxInsertRequiresExistingEvent`.

---

### 13. BatchAppendEvents does not validate all events belong to the same campaign
**Category:** correctness risk
**File:** `internal/services/game/storage/sqlite/eventjournal/store_events_append.go:166-197`

The doc comment states "All events must belong to the same campaign," but the implementation only reads the campaign ID from `validated[0].CampaignID` and uses it for all events. If a caller passes events with mixed campaign IDs, events beyond the first would be stored under the wrong campaign's sequence counter and chain. The chain hash would still be computed correctly for the assumed campaign, but the `campaign_id` column in the row would reflect the individual event's ID while the sequence comes from the first event's campaign.

Actually, looking more carefully: the `qtx.AppendEvent` call at line 260 uses `campaignID` (from `validated[0]`) for the INSERT, so all rows get the first event's campaign ID regardless of their actual `CampaignID` field. This is correct if the caller always passes same-campaign events, but a mixed-campaign batch would silently misattribute events.

**Proposal:** Add an explicit validation loop checking `validated[i].CampaignID == campaignID` for all events before opening the transaction.

---

### 14. Integration outbox store_test.go duplicates `testKeyring` helper
**Category:** contributor friction (minor)
**File:** `internal/services/game/storage/sqlite/integrationoutbox/store_test.go:149-159` and `internal/services/game/storage/sqlite/eventjournal/store_test_helpers_test.go:17-27`

Both test files define identical `testKeyring` helper functions. This is a consequence of Go test package boundaries (external test vs internal test), but contributes to maintenance duplication.

**Proposal:** Consider extracting a `testutil` or `storagetest` package with shared test helpers. Alternatively, accept this as normal Go test hygiene for separate packages.

---

### 15. `listEventCampaignIDs` uses raw SQL instead of sqlc-generated query
**Category:** missing best practice (minor)
**File:** `internal/services/game/storage/sqlite/eventjournal/store_events_integrity.go:46-64`

The `listEventCampaignIDs` helper uses a raw `s.sqlDB.QueryContext` call with `"SELECT DISTINCT campaign_id FROM events ORDER BY campaign_id"`. This is a simple, safe query, but it diverges from the pattern of using `s.q.*` for all event queries. It's likely not in the sqlc definitions because it's only used internally for integrity verification.

**Proposal:** Add a sqlc query definition for this. Low priority since the raw query is trivially correct.

---

### 16. ListVisibleOpenScenesForCharacters builds dynamic IN clause with fmt.Sprintf
**Category:** missing best practice
**File:** `internal/services/game/storage/sqlite/coreprojection/store_projection_scene_record.go:243-257`

The method builds an `IN (?, ?, ...)` clause using `fmt.Sprintf` with `strings.Join(placeholders, ",")`. All values are properly parameterized -- the only interpolated content is the `?` placeholder count. This is safe. However, there's no upper bound on the number of character IDs, which could create extremely long SQL statements.

**Proposal:** Add a practical limit (e.g., 100 character IDs) or document that callers must provide bounded input. SQLite has a `SQLITE_MAX_VARIABLE_NUMBER` limit (default 999) that would eventually fail for very large lists.

---

### 17. Migration numbering collision in projections
**Category:** contributor friction
**File:** `internal/services/game/storage/sqlite/migrations/projections/`

Migrations `003_campaign_cover_asset.sql` and `003_projection_apply_checkpoints.sql` share the same numeric prefix `003_`. Similarly, `011_daggerheart_profile_heritage_companion.sql` and `011_scene_projections.sql` share prefix `011_`. The migration framework sorts lexicographically, so `003_campaign_cover_asset.sql` runs before `003_projection_apply_checkpoints.sql` because `c` < `p`. This works deterministically but makes the intended order ambiguous to contributors.

**Proposal:** Use unique numeric prefixes for all migration files. When two migrations are independent and can run in either order, use sequential numbers anyway to make ordering explicit.

---

### 18. coreprojection Store exposes DB-level transaction via txStore
**Category:** (positive finding)
**File:** `internal/services/game/storage/sqlite/coreprojection/store.go:27-35`

The `txStore` method creates a shallow clone with a transaction-scoped query bundle, enabling exactly-once projection apply to run the apply callback inside the checkpoint transaction. This is a clean approach that avoids exposing raw `*sql.Tx` to callers.

---

### 19. Outbox retry backoff uses exponential with cap
**Category:** (positive finding)
**File:** `internal/services/game/storage/sqlite/projectionapplyoutbox/store.go:355-364`

`retryBackoff` uses `time.Second << (attempt - 1)` capped at 5 minutes. This gives 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s, 300s backoff schedule. The dead-letter threshold at 8 attempts means maximum total wait is bounded. Clean implementation.

---

### 20. Integrity package delegates to domain event package for hash computation
**Category:** (positive finding)
**File:** `internal/services/game/storage/integrity/event_hash.go:11-21`

`integrity.EventHash` and `integrity.ChainHash` both delegate to `event.EventHash` and `event.ChainHash`, ensuring canonical field ordering is defined in exactly one place. The `TestHashParityWithDomainPackage` test guards against drift. This is a strong anti-drift pattern.

---

### 21. HKDF key derivation per campaign is a strong isolation pattern
**Category:** (positive finding)
**File:** `internal/services/game/storage/integrity/keyring.go:91-101`

Each campaign gets its own derived HMAC key via HKDF with `"campaign:"+campaignID` as info. This means compromising one campaign's chain signatures doesn't reveal keys for other campaigns, even though all derive from the same root key.

---

### 22. `DB()` method on eventjournal Store exposes raw *sql.DB
**Category:** anti-pattern (minor)
**File:** `internal/services/game/storage/sqlite/eventjournal/store_integration_outbox_provider.go:12-17`

The `DB()` method exposes the raw `*sql.DB` handle for sibling backend packages. This is used by `IntegrationOutboxStore()` and `ProjectionApplyOutboxStore()` to bind their backends. While documented as intentional for "sibling backend packages," it also enables test code to issue arbitrary SQL against the event database (used heavily in outbox tests). This is pragmatic but reduces encapsulation.

**Proposal:** Accept as-is. The alternative (passing `*sql.DB` through constructors) would complicate the `Open` function's API. The sibling-binding pattern is well-documented.

---

## Architecture Assessment

The storage layer demonstrates several strong architectural qualities:

1. **Clean contract boundaries**: Six contract files organize 20+ interfaces by domain concern (campaign/participant/character, events/audit/stats, outboxes, projection apply, projection state, session/scene). The Reader/Store split is applied consistently to all projection stores.

2. **Purpose-scoped composites**: `CampaignReadStores`, `SessionReadStores`, `SceneReadStores` group related stores for transaction-scoped and handler-scoped consumption without creating a god interface.

3. **Append-only event integrity**: Database triggers, hash chains, HMAC signatures, per-campaign key derivation, and startup verification create defense-in-depth for the event journal.

4. **Outbox atomicity**: Both outboxes (projection-apply and integration) enqueue inside the event-append transaction, guaranteeing no silent gaps between events and their downstream work items.

5. **Well-separated SQLite backends**: `eventjournal`, `coreprojection`, `integrationoutbox`, `projectionapplyoutbox`, and `daggerheartprojection` each own a focused responsibility within the SQLite backend family.

The naming of `CampaignReadStores` (Finding 8) is the most impactful architectural concern, as it suggests read-only access while actually including write methods.
