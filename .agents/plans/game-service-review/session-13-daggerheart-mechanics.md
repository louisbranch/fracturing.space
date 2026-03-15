# Session 13: Daggerheart Domain — Mechanics, Content, Internals

## Status: `complete`

## Package Summaries

### `domain/bridge/daggerheart/internal/mechanics/` (11 files)
Internal mechanics engine: damage calculation, probability, hit point tracking. Pure domain logic with no external dependencies.

### `domain/bridge/daggerheart/internal/projection/` (5 files)
System-specific projection logic for Daggerheart read models.

### `domain/bridge/daggerheart/internal/reducer/` (3 files)
State reducers for complex Daggerheart state transitions.

### `domain/bridge/daggerheart/domain/` (13 files)
Daggerheart-specific domain types: character profiles, conditions, abilities, etc.

### `domain/bridge/daggerheart/content/`, `contentstore/`, `projectionstore/`, `profile/`
Content management and storage contracts for Daggerheart game content.

## Findings

### Finding 1: internal/mechanics/ Is Properly Isolated
- **Severity**: info
- **Location**: `domain/bridge/daggerheart/internal/mechanics/`
- **Issue**: Pure game mechanics calculations (damage, probability, HP) isolated from infrastructure. This is testable without storage or transport dependencies.
- **Recommendation**: Clean isolation. Good example of separating game logic from system plumbing.

### Finding 2: internal/projection/ — System-Specific Projection Is Correctly Placed
- **Severity**: info
- **Location**: `domain/bridge/daggerheart/internal/projection/`
- **Issue**: System-specific projections belong inside the system module, not in the core `projection/` package. This keeps Daggerheart read-model logic co-located with the system that defines the events.
- **Recommendation**: Correct placement.

### Finding 3: domain/ Sub-Package Naming Collision
- **Severity**: medium
- **Location**: `domain/bridge/daggerheart/domain/`
- **Issue**: Having a `domain/` sub-package inside `domain/bridge/daggerheart/` creates a naming collision with the parent `domain/` layer. Import paths become confusing: `domain/bridge/daggerheart/domain` vs top-level `domain/`. Contributors may confuse the scope.
- **Recommendation**: Rename to `daggerheart/types/` or `daggerheart/model/` to avoid the `domain` name collision.

### Finding 4: contentstore/ and projectionstore/ Inside Domain — Architectural Smell
- **Severity**: high
- **Location**: `domain/bridge/daggerheart/contentstore/`, `domain/bridge/daggerheart/projectionstore/`
- **Issue**: Storage contracts (`contentstore/`, `projectionstore/`) are defined inside the domain layer. This inverts the dependency direction — domain packages should not define storage interfaces. Storage contracts should live in the `storage/` package or at the system module boundary.
- **Recommendation**: Move storage contracts to `storage/` (e.g., `storage/contracts_daggerheart.go`) or define them as interfaces in the bridge/module boundary layer. The domain sub-packages should contain pure domain types only.

### Finding 5: Probability Module Testing
- **Severity**: info
- **Location**: `domain/bridge/daggerheart/internal/mechanics/`
- **Issue**: Probability calculations in the mechanics package need deterministic testing with fixed random sources. The injectable randomness from `core/random/` supports this.
- **Recommendation**: Verify that probability tests use deterministic seeds.

### Finding 6: Top-Level Damage/Death/Downtime/Rest Files vs internal/mechanics/
- **Severity**: medium
- **Location**: `domain/bridge/daggerheart/damage*.go`, `death.go`, `downtime.go`, `rest*.go` vs `internal/mechanics/`
- **Issue**: There are damage/recovery-related files at the top level (decider-facing) AND in `internal/mechanics/` (calculation logic). The boundary should be: top-level files are decider cases that delegate to `internal/mechanics/` for computation. If top-level files contain computation logic, it should be extracted into mechanics.
- **Recommendation**: Ensure top-level files are thin decider wrappers that delegate to `internal/mechanics/` for all computation. Extract any duplicated logic.

## Summary Statistics
- Files reviewed: ~47 (11 mechanics + 5 projection + 3 reducer + 13 domain + 15 content/store/profile)
- Findings: 6 (0 critical, 1 high, 2 medium, 0 low, 3 info)
