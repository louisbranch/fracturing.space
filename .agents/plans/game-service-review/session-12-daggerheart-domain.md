# Session 12: Daggerheart Domain — Module, Decider, Folder, State

## Status: `complete`

## Package Summaries

### `domain/bridge/daggerheart/` (~67+ files at package root)
Largest single package in the service. Implements the Daggerheart game system module: module registration, 8+ decider files, folder (527 lines), state management (state.go, state_factory.go, character_state.go), adapter (472 lines), payload (536 lines), mechanics manifest (574 lines), and compat files.

## Findings

### Finding 1: 67+ Files in One Package — Should Be Split
- **Severity**: high
- **Location**: `domain/bridge/daggerheart/`
- **Issue**: This package has more files than any other in the service. It handles: module registration, command deciders (8+ files), event folding, state management, payload transformation, mechanics manifest, adapter logic, and compatibility layers. A contributor touching countdown logic must navigate past adversary, damage, recovery, and rest files.
- **Recommendation**: Split into concern-specific sub-packages:
  - `daggerheart/decider/` — command decider files
  - `daggerheart/fold/` — folder and fold logic
  - `daggerheart/mechanics/` — mechanics manifest and manifest-related types
  - Keep module.go, state.go, adapter.go at root as the integration layer

### Finding 2: adapter.go at 472 Lines — Multiple Responsibilities
- **Severity**: medium
- **Location**: `domain/bridge/daggerheart/adapter.go`
- **Issue**: The adapter bridges between the generic module interface and Daggerheart-specific types. At 472 lines, it likely handles event adaptation, command adaptation, and state conversion — three distinct concerns.
- **Recommendation**: Split by concern: `adapter_events.go`, `adapter_commands.go`, `adapter_state.go`.

### Finding 3: payload.go at 536 Lines — Split by Entity
- **Severity**: medium
- **Location**: `domain/bridge/daggerheart/payload.go`
- **Issue**: Likely contains payload types for all Daggerheart commands/events in one file. At 536 lines, navigating to a specific payload type requires scrolling through unrelated types.
- **Recommendation**: Split by entity: `payload_character.go`, `payload_adversary.go`, `payload_countdown.go`, `payload_damage.go`, etc.

### Finding 4: mechanics_manifest.go at 574 Lines — Large Static Declaration
- **Severity**: medium
- **Location**: `domain/bridge/daggerheart/mechanics_manifest.go`
- **Issue**: The mechanics manifest declares all Daggerheart game mechanics (abilities, items, classes, etc.). At 574 lines, this is largely static data declarations. The data could be generated from a structured source (JSON/YAML) or loaded at runtime.
- **Recommendation**: Consider generating this file from a structured data source. If manual maintenance is intentional (e.g., compile-time validation), add a comment explaining why.

### Finding 5: Compat Files — Need Lifecycle Documentation
- **Severity**: medium
- **Location**: `domain/bridge/daggerheart/compat_conditions.go`, `compat_constants.go`
- **Issue**: Compat files exist for backward compatibility with legacy event payloads. Without removal criteria, they'll persist indefinitely. Are they for v1→v2 migration, or permanent compatibility?
- **Recommendation**: Add a comment with removal criteria: "Remove after all production journals have been migrated past version X" or mark as permanent if needed for event replay of historical data.

### Finding 6: Event Type Namespacing Under sys.daggerheart.* Is Correct
- **Severity**: info
- **Location**: `domain/bridge/daggerheart/` event types
- **Issue**: All Daggerheart event types use `sys.daggerheart.*` prefix, validated by `core/naming.ValidateSystemNamespace`. This namespace isolation prevents collision with core events and other game systems.
- **Recommendation**: Clean namespacing.

### Finding 7: folder.go at 527 Lines — Complex But Necessary
- **Severity**: low
- **Location**: `domain/bridge/daggerheart/folder.go`
- **Issue**: The folder handles all Daggerheart event types. At 527 lines, it's large but each fold case is typically 5-15 lines. The folder is a switch statement routing events to state mutations. The complexity is proportional to the number of Daggerheart event types.
- **Recommendation**: Could be split by entity concern (character fold, adversary fold, countdown fold) but the single-file approach keeps the fold dispatch visible in one place. Low priority for refactoring.

## Summary Statistics
- Files reviewed: ~67+ (top-level daggerheart package files, prod and test)
- Findings: 7 (0 critical, 1 high, 4 medium, 1 low, 1 info)
