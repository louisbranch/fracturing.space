# Session 10: Cross-Aggregate Workflows

## Status: `complete`

## Package Summaries

### `domain/readiness/` (~13 files)
Cross-aggregate coordinator checking campaign readiness for operations (session start, game system binding). Queries multiple aggregate states without modifying them.

### `domain/campaignbootstrap/` (~3 files)
Cross-aggregate command: creates campaign + initial participant atomically. The only command that spans multiple aggregates in a single decision.

### `domain/fork/` (~4 files)
Campaign forking (deep copy) workflow. Creates a new campaign from existing history.

### `domain/replay/` (~4 files)
Event replay infrastructure: checkpoint management, replay pipeline types, replay execution.

### `domain/journal/` (~4 files)
In-memory event journal implementation for testing.

### `domain/checkpoint/` (~6 files)
Checkpoint store abstractions and in-memory implementation.

### `domain/internal/` and `domain/internaltest/`
Internal shared helpers and architecture contract tests.

## Findings

### Finding 1: Readiness Is Properly Scoped as Cross-Aggregate Coordinator
- **Severity**: info
- **Location**: `domain/readiness/`
- **Issue**: Readiness checks query campaign, participant, session, and character states to determine if an operation can proceed. It reads but never writes. This is the correct pattern for a cross-aggregate coordinator — it's a read-only query, not a saga or process manager.
- **Recommendation**: Clean design. The coordinator pattern avoids cross-aggregate mutations.

### Finding 2: CampaignBootstrap as Only Cross-Aggregate Command — Appropriate
- **Severity**: info
- **Location**: `domain/campaignbootstrap/`
- **Issue**: Bootstrap creates campaign + owner participant atomically. This is the only cross-aggregate command because it's the only operation where two entities must be created together (a campaign without an owner is invalid). Other operations use eventual consistency via events.
- **Recommendation**: Correct constraint. Adding more cross-aggregate commands should be resisted.

### Finding 3: journal/ and checkpoint/ In-Memory Implementations Are Test Doubles
- **Severity**: info
- **Location**: `domain/journal/`, `domain/checkpoint/`
- **Issue**: Both packages provide in-memory implementations of the journal and checkpoint store interfaces. These are test doubles — production uses SQLite implementations in `storage/sqlite/`. The in-memory versions are used by unit/integration tests to avoid database dependencies.
- **Recommendation**: Correct placement. In-memory implementations in the domain layer keep tests fast and dependency-free.

### Finding 4: replay/ vs aggregate/ Fold Boundary Is Clear
- **Severity**: info
- **Location**: `domain/replay/`, `domain/aggregate/folder.go`
- **Issue**: `replay/` handles the replay pipeline: loading events from journal, resuming from checkpoints, feeding events through the folder. `aggregate/Folder` handles the actual state fold (event → state transition). Replay orchestrates, folder executes. Different responsibilities at different levels.
- **Recommendation**: Clean boundary.

### Finding 5: Architecture Tests in domain/internaltest/ Validate Structural Invariants
- **Severity**: info
- **Location**: `domain/internaltest/`
- **Issue**: Architecture tests verify structural constraints like: no import cycles, proper package layering, module conformance. These are run at build time to catch structural regressions.
- **Recommendation**: Good investment in architecture tests. These protect the codebase as it grows.

### Finding 6: fork/ May Need Stronger Boundary Documentation
- **Severity**: low
- **Location**: `domain/fork/`
- **Issue**: Campaign forking is a complex operation that deep-copies event history. The boundary between what gets forked (events) and what gets reset (IDs, timestamps) should be clearly documented for contributors.
- **Recommendation**: Add a "forking semantics" document explaining which state transfers and which resets.

## Summary Statistics
- Files reviewed: ~38 (13 readiness + 3 bootstrap + 4 fork + 4 replay + 4 journal + 6 checkpoint + 4 internal/internaltest)
- Findings: 6 (0 critical, 0 high, 0 medium, 1 low, 5 info)
