# Session 17: Projection Layer

## Status: `complete`

## Package Summaries

### `projection/` (~21 prod files, ~9,474 LOC total)
Read-model projection layer. Key components:
- `applier.go` (142 lines) — Projection applier orchestrating event-to-store writes
- `applier_event_preprocess.go`, `applier_watermark.go` — Pre-processing and watermark tracking
- `apply_*.go` (8 entity-specific files) — Per-entity projection handlers
- `apply_interaction.go` (531 lines) — Largest apply file
- `core_router.go` (85 lines) — Routes events to handlers
- `handler_registry.go` (345 lines) — Registry of projection handlers
- `replay.go`, `replay_contracts.go` — Projection replay logic
- `gaps.go`, `campaign_status.go`, `parse.go` — Supporting utilities

## Findings

### Finding 1: Applier Design Is Clean — Event-to-Store Orchestration
- **Severity**: info
- **Location**: `projection/applier.go`
- **Issue**: The applier is the projection counterpart to the domain folder. It routes events to entity-specific handlers that write to storage. The applier handles: event preprocessing, watermark tracking, handler dispatch, and gap detection. At 142 lines, it's well-scoped.
- **Recommendation**: Clean design.

### Finding 2: apply_interaction.go at 531 Lines — Split Candidate
- **Severity**: high
- **Location**: `projection/apply_interaction.go`
- **Issue**: Interaction projection handles gates, spotlights, OOC pauses, and AI turns — the same multi-concern problem as the session domain aggregate. This is the largest apply file and mirrors the session complexity.
- **Recommendation**: Split into: `apply_interaction_gate.go`, `apply_interaction_spotlight.go`, `apply_interaction_ooc.go`, `apply_interaction_ai_turn.go`. This mirrors the session domain split recommendation.

### Finding 3: Handler Registry Is Separate from Engine Registries — Correct
- **Severity**: info
- **Location**: `projection/handler_registry.go` (345 lines)
- **Issue**: The projection handler registry maps event types to projection handlers. This is separate from the engine's command/event registries because projection handles different concerns (read model updates vs write validation). The startup validation (Session 8) verifies that all projection-intent events have handlers.
- **Recommendation**: Correct separation. The validation chain links them.

### Finding 4: Replay Boundary with domain/replay/ Is Clear
- **Severity**: info
- **Location**: `projection/replay.go`, `domain/replay/`
- **Issue**: `domain/replay/` handles aggregate state reconstruction (fold). `projection/replay.go` handles read-model reconstruction (re-projecting events through handlers). Different purposes: write-model vs read-model replay.
- **Recommendation**: Clean boundary.

### Finding 5: applier_test.go at 3,940 Lines — Maintainability Concern
- **Severity**: medium
- **Location**: `projection/applier_test.go`
- **Issue**: 3,940 lines of test code for the applier. This likely includes comprehensive event-by-event projection tests for all entity types. While thorough, a single test file this large is hard to navigate and maintain.
- **Recommendation**: Split into per-entity test files: `apply_campaign_test.go`, `apply_participant_test.go`, etc. This aligns with the production code split.

### Finding 6: Projection Parse Helpers
- **Severity**: info
- **Location**: `projection/parse.go`
- **Issue**: Parse helpers extract payload fields from events for projection. This is the projection counterpart to decider payload unmarshaling.
- **Recommendation**: Clean pattern.

## Summary Statistics
- Files reviewed: ~35 (21 production + 14 test)
- Findings: 6 (0 critical, 1 high, 1 medium, 0 low, 4 info)
