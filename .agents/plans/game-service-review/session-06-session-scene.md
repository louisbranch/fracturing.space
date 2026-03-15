# Session 6: Core Aggregates — Session and Scene

## Status: `complete`

## Package Summaries

### `domain/session/` (22 prod files, ~5,875 LOC total)
Largest core aggregate. Manages: session lifecycle (start/end), gates (open/respond/resolve/abandon), spotlights, active scene tracking, OOC pauses, AI turn orchestration, and GM authority. Registry at 648 lines registers all session event types. Gate workflow subsystem spans 8 files (`gate_progress_*`, `gate_projection_*`, `gate_workflow_*`).

### `domain/scene/` (14+ prod files, ~3,676 LOC total)
Scene aggregate managing narrative scope with character rosters, gates, spotlights, player phases, and GM output. Scene state is 20+ fields with player phase sub-state.

## Findings

### Finding 1: Session at 22 Production Files — Multiple Sub-Concerns
- **Severity**: high
- **Location**: `domain/session/`
- **Issue**: The session aggregate handles 5+ distinct behavioral domains: lifecycle (start/end), gates (8 files for workflow+projection+progress), spotlights, OOC pauses, and AI turns. This is the most complex aggregate in the system. While each concern is split into its own file(s), they all fold into a single `session.State` with 22 fields.
- **Recommendation**: Consider whether gates, OOC pauses, or AI turns could become their own lightweight aggregates or at least use sub-state structs. The gate subsystem alone (8 files) warrants a `session/gate/` sub-package. This would reduce the cognitive load for contributors touching one concern without affecting others.

### Finding 2: session/registry.go at 648 Lines — Registering Cross-Concern Events
- **Severity**: high
- **Location**: `domain/session/registry.go`
- **Issue**: The session registry registers events for gates, spotlights, OOC, AI turns, interactions, and lifecycle — all in one file. At 648 lines, it's the largest registry file and registers events that arguably belong to distinct behavioral domains. A contributor adding a new gate event type must navigate past OOC and AI turn registrations.
- **Recommendation**: Split the registry into concern-specific files: `registry_lifecycle.go`, `registry_gate.go`, `registry_spotlight.go`, `registry_ooc.go`, `registry_ai_turn.go`. Each file registers its own event types and returns them from a sub-function that the main registry aggregates.

### Finding 3: SpotlightType Defined in Both Session and Scene
- **Severity**: medium
- **Location**: `domain/session/state.go:28`, `domain/scene/state.go:44`
- **Issue**: Both `session.State` and `scene.State` have `SpotlightType` fields. Session uses `string` type, scene uses `SpotlightType` (a named type). The types are structurally compatible but defined differently. This creates confusion about which is canonical and whether they share the same value space.
- **Recommendation**: Define `SpotlightType` once in a shared location (e.g., `domain/session/` or a shared types file) and use it consistently in both session and scene state. If session and scene spotlights have different semantics, document why they're separate.

### Finding 4: Gate Subsystem Spans 8 Files — Sub-Package Candidate
- **Severity**: medium
- **Location**: `domain/session/gate_*.go` (8 files)
- **Issue**: Gate files: `gate_progress_api.go`, `gate_progress_state.go`, `gate_progress_types.go`, `gate_projection_json_helpers.go`, `gate_projection_metadata.go`, `gate_projection_progress.go`, `gate_projection_resolution.go`, `gate_workflow_api.go`, `gate_workflow_base.go`, `gate_workflow_generic.go`, `gate_workflow_helpers.go`. This is a full subsystem with its own state, types, workflow logic, and projection helpers.
- **Recommendation**: Extract into `domain/session/gate/` sub-package. The gate concern has clear boundaries and its own conceptual API.

### Finding 5: Session State AI Turn Fields Are a Sub-Aggregate
- **Severity**: medium
- **Location**: `domain/session/state.go:41-54`
- **Issue**: 8 fields dedicated to AI turn tracking: `AITurnStatus`, `AITurnToken`, `AITurnOwnerParticipantID`, `AITurnSourceEventType`, `AITurnSourceSceneID`, `AITurnSourcePhaseID`, `AITurnLastError`. These form a distinct lifecycle (queued→started→completed/failed) with their own invariants.
- **Recommendation**: Group into `AITurnState` sub-struct: `AITurn AITurnState` on `session.State`. This reduces field count and makes the AI turn concern self-documenting.

### Finding 6: Scene State Has Player Phase Sub-State — Already Well-Structured
- **Severity**: info
- **Location**: `domain/scene/state.go:27-64`
- **Issue**: Scene state includes player phase tracking (7 fields + PlayerPhaseSlot map). Unlike session's flat AI turn fields, the player phase sub-state is already somewhat structured with a separate `PlayerPhaseSlot` type. However, the player phase fields are still flat on the main State struct.
- **Recommendation**: Consider a `PlayerPhaseState` sub-struct similar to the AI turn recommendation for session.

### Finding 7: Duplication Between Session and Scene Gates
- **Severity**: medium
- **Location**: `domain/session/state.go:19-27`, `domain/scene/state.go:39-42`
- **Issue**: Both session and scene have `GateOpen`, `GateID`, and (session only) `GateType`/`GateMetadataJSON`. Scene gates have fewer metadata fields. The gate workflow files in session handle both session and scene gates through generic abstractions, but the state duplication means gate behavior must be validated independently at both levels.
- **Recommendation**: Extract a shared `GateState` struct with common fields. Scene and session embed it and add level-specific fields. This ensures gate invariants are defined once.

### Finding 8: Scene HasPC Method Is a Query on Fold State
- **Severity**: info
- **Location**: `domain/scene/state.go:68-75`
- **Issue**: `HasPC(pcs map[ids.CharacterID]bool)` is a query method on state, not a fold or decider. This is appropriate — state can expose query helpers for deciders.
- **Recommendation**: Clean pattern.

## Summary Statistics
- Files reviewed: ~59 (33 session + 26 scene, prod and test)
- Findings: 8 (0 critical, 2 high, 4 medium, 0 low, 2 info)
