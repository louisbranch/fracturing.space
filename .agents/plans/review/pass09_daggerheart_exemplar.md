# Pass 9: Daggerheart System as Game System Author API Exemplar

## Summary

The Daggerheart system module is a mature, well-architected reference implementation
with 199 Go files across 19 sub-packages. The module interface (`module.Module`) is
clean and the manifest wiring (`manifest.go`) provides a clear single-descriptor
pattern for system registration. The main areas of concern for second-system authors
are: (1) the heavy export-alias layer needed to test `internal/` packages from the
root, (2) duplicated type conversions between domain, projection, and storage
boundaries, (3) a `RegistrySystem` that intentionally returns nil for both
`StateHandlerFactory` and `OutcomeApplier`, making the metadata registry bridge
partially dead code, and (4) the sheer scale of boilerplate a second system would
need to replicate vs. what the framework actually provides as reusable infrastructure.

Overall the system is well-organized with clear package responsibilities, thorough
doc.go annotations, and a reading-order guide in the root doc.go. The architecture
is sound enough to serve as a template -- the findings below are refinements, not
structural objections.

---

## Findings

### 1. `internal/decider/exports.go` is a large export-alias file for root-package testing

**Category:** contributor friction / anti-pattern
**File:** `internal/services/game/domain/systems/daggerheart/internal/decider/exports.go`
**Lines:** 1-137

The `exports.go` file re-exports 34 constants, 16 variables (function aliases), the
`DecisionHandler` type alias, and the `DecisionHandlers` map. This pattern exists
solely because decider logic lives under `internal/` but the 33 test files sit in the
root `daggerheart` package. The test files import `internal/decider` through the
parent package's `internal/` access, but need exported symbols to exercise behavior.

This creates a maintenance tax: every new command type, rejection code, or helper
function requires a parallel export line. A second system author would need to
replicate this entire pattern or find a different testing strategy.

**Proposal:** Consider one of:
- Move decider tests into the `internal/decider` package itself (they can test
  unexported functions directly). The root package tests would then test only the
  module-level seams.
- Flatten the decider out of `internal/` into a sibling package like
  `daggerheart/decider` that is technically importable but convention-guarded.
  This eliminates the export alias layer entirely.

---

### 2. `internal/folder/exports.go` has the same pattern on a smaller scale

**Category:** contributor friction
**File:** `internal/services/game/domain/systems/daggerheart/internal/folder/exports.go`
**Lines:** 1-29

Three fold functions are re-exported for root-package seam tests. Same concern as
finding #1 but smaller scope.

**Proposal:** Same as #1 -- either move tests into the folder package or promote
the package out of `internal/`.

---

### 3. Dual `StatePatch` type definitions across internal/adapter and internal/projection

**Category:** missing best practice (DRY)
**Files:**
- `internal/services/game/domain/systems/daggerheart/internal/adapter/adapter.go:21-32`
- `internal/services/game/domain/systems/daggerheart/internal/projection/character_state.go:89-103`

Both packages define a `StatePatch` struct with identical field sets (HP, Hope,
HopeMax, Stress, Armor, LifeState, ClassState, SubclassState, CompanionState,
ImpenetrableUsedThisShortRest). The adapter version references
`daggerheartstate.CharacterClassState` while the projection version references
`projectionstore.DaggerheartClassState`, so they are not type-identical -- but
they represent the same concept.

**Proposal:** Extract a shared `StatePatch` type into `state/` or `reducer/`
parameterized on the class/subclass state types (or define a single canonical one
and convert at boundaries).

---

### 4. RegistrySystem returns nil for StateHandlerFactory and OutcomeApplier

**Category:** anti-pattern / dead code smell
**File:** `internal/services/game/domain/systems/daggerheart/registry_system.go:42-52`

```go
func (r *RegistrySystem) StateHandlerFactory() bridge.StateHandlerFactory {
    return nil
}
func (r *RegistrySystem) OutcomeApplier() bridge.OutcomeApplier {
    return nil
}
```

The `GameSystem` interface in `registry_bridge.go` declares these methods, and
`RegistrySystem` explicitly returns nil with TODO-style comments. This means the
entire `StateHandlerFactory` / `OutcomeApplier` / `CharacterStateHandler` /
`SnapshotStateHandler` / `Healable` / `Damageable` / `ResourceHolder` interface
hierarchy defined in `registry_bridge.go:126-225` is currently unimplemented by the
only existing system. A second system author would face an unclear signal: implement
these interfaces or skip them as Daggerheart does?

**Proposal:** Either:
- Implement the bridge handlers in Daggerheart (the `mechanics.CharacterState`
  already satisfies `Healable`, `Damageable`, and `ResourceHolder` -- wiring is
  straightforward).
- Or remove the interfaces from the `GameSystem` contract until they have a real
  consumer, to avoid misleading second-system authors into implementing dead code.

---

### 5. ToStorage / FromStorage conversion boilerplate in character_profile.go

**Category:** contributor friction
**File:** `internal/services/game/domain/systems/daggerheart/state/character_profile.go:345-545`

The `ToStorage()` method (100 lines) and `CharacterProfileFromStorage()` function
(95 lines) are field-by-field manual conversions between `state.CharacterProfile`
and `projectionstore.DaggerheartCharacterProfile`. These types have near-identical
field sets (40+ fields). Adding a single field requires changes in both types plus
both conversion functions -- four touch points.

A second system would face the same problem: defining profile types in both `state/`
and `projectionstore/` packages and maintaining manual converters.

**Proposal:** Consider:
- Code generation for projection store types from the canonical domain type.
- A single shared profile type with package-specific type aliases.
- Or at minimum, a struct-mapping helper to reduce line count.

---

### 6. Condition and countdown state types duplicated across three packages

**Category:** missing best practice (DRY)
**Files:**
- `state/snapshot_state.go` -- `AdversaryState`, `EnvironmentEntityState`, `CountdownState`
- `projectionstore/contracts.go` -- `DaggerheartAdversary`, `DaggerheartEnvironmentEntity`, `DaggerheartCountdown`
- `rules/` -- `AdversaryFeatureState`, `Countdown`, etc.

The same logical entity (e.g., countdown) has distinct struct definitions in `state/`,
`projectionstore/`, and `rules/`, with manual conversion between them. This is
intentional (write-path state vs. read-path projection vs. pure rules) but the
conversion overhead is significant.

**Proposal:** Document the three-type pattern in the game-systems architecture doc
as an expected cost so second-system authors budget for it, or investigate whether
the projection store types can be the canonical source.

---

### 7. No tests exist inside `internal/decider/`, `internal/folder/`, `internal/validator/`

**Category:** contributor friction
**Evidence:** `find internal/decider -name "*_test.go" | wc -l` = 0

All 33 test files for decider logic live in the root `daggerheart` package. The
`internal/folder/` has no test files either (the single `projection_test.go` is in
`internal/projection/`). The `internal/validator/` has zero test files.

This means:
- Decider/folder/validator logic cannot be tested in isolation without the
  export-alias layer (finding #1).
- A contributor changing one decider handler must understand the root package test
  infrastructure to write or modify tests.
- Coverage attribution is blurred -- `go test ./daggerheart/...` covers internal
  packages indirectly.

**Proposal:** Add focused unit tests inside the `internal/` packages for pure logic,
keeping root-level tests for integration seams only. This would also reduce the
need for the exports.go files.

---

### 8. 19 sub-packages may be excessive for the module interface surface

**Category:** contributor friction
**Evidence:** `find ... -type d | wc -l` = 19

The Daggerheart system has 19 directories:
root, content/, content/filter/, contentstore/, countdowns/, domain/, internal/,
internal/adapter/, internal/decider/, internal/folder/, internal/projection/,
internal/reducer/, internal/validator/, mechanics/, payload/, profile/,
projectionstore/, rules/, state/

The module interface itself (`module.Module`) requires only 7 methods. The sub-package
explosion serves real separation concerns (rules vs. mechanics vs. state vs. payload)
but a second-system author would need a map to understand what goes where.

**Proposal:** The root `doc.go` reading order is a good start. Extend it to a
formal "second system onboarding" section in `docs/architecture/systems/game-systems.md`
that maps each package responsibility to the module interface method it supports:
- "Your `Decider()` implementation goes in a decider package"
- "Your `Folder()` implementation goes in a folder package"
- "Define payload types in a `payload/` package"
- etc.

---

### 9. `commandids` package introduces a third location for command type strings

**Category:** contributor friction
**File:** `internal/services/game/domain/systems/daggerheart/internal/decider/decider.go:14-52`

Command type constants are defined in three places:
1. `domain/commandids/` -- canonical string constants (e.g., `DaggerheartGMMoveApply`)
2. `internal/decider/decider.go` -- package-level unexported constants aliasing commandids
3. `internal/decider/exports.go` -- exported re-aliases of the unexported constants

A second system would need to add entries to `commandids/` and then create the
local alias chain. The three-hop indirection is confusing.

**Proposal:** Systems should use `commandids.DaggerheartGMMoveApply` directly in
the decider handler map and module registration. The local aliases provide no type
safety (they are all `command.Type` = `string`) and exist only for shorter names.

---

### 10. `LevelUpApplier` dependency injection creates an unusual wiring pattern

**Category:** contributor friction
**Files:**
- `internal/services/game/domain/systems/daggerheart/folder.go:7-9`
- `internal/services/game/domain/systems/daggerheart/adapter.go:10-12`
- `internal/services/game/domain/systems/daggerheart/internal/folder/folder.go:12-13`
- `internal/services/game/domain/systems/daggerheart/internal/adapter/adapter.go:16`

Both `Folder` and `Adapter` receive a `LevelUpApplier` function at construction,
injected from the root package's `applyLevelUpToCharacterProfile`. This is documented
as avoiding a circular dependency (internal packages cannot import root). However,
the pattern is unusual -- it creates a function-injection seam for a single helper.

A second system with similar cross-cutting profile mutations would need to discover
and replicate this pattern.

**Proposal:** Document this DI pattern in the game-systems architecture doc, or
consider moving `applyLevelUpToCharacterProfile` into a shared internal package
(e.g., `mechanics/` or `profile/`) that both folder and adapter can import directly.

---

### 11. `rest_package.go` is a 471-line orchestration function in the root package

**Category:** missing best practice
**File:** `internal/services/game/domain/systems/daggerheart/rest_package.go:1-471`

`ResolveRestPackage` is a large orchestration function that combines rest resolution,
downtime move resolution, countdown mutations, and participant normalization. It
lives in the root `daggerheart` package rather than in `mechanics/` or `rules/`
because it depends on `payload/`, `rules/`, `countdowns/`, and `mechanics/`
simultaneously. The function has 8 downtime move branches, each with their own
validation and state mutation logic.

**Proposal:** Consider extracting the downtime-move resolution switch into a
strategy pattern in `mechanics/` or `rules/`, keeping the orchestration shell in
the root package but reducing its line count. This would also make individual
downtime moves independently testable.

---

### 12. Root package facade files re-export types/functions from sub-packages

**Category:** contributor friction
**Files:**
- `character_state.go` -- re-exports `mechanics.ResourceHope`, `CharacterStateConfig`, etc.
- `death.go` -- re-exports `mechanics.DeathMoveBlazeOfGlory`, etc.
- `downtime.go` -- re-exports `mechanics.DowntimeClearAllStress`, etc.
- `rest.go` -- re-exports `mechanics.RestTypeShort`, etc.

These facade files exist for backward compatibility and convenience, but they add
maintenance overhead and can mislead contributors into thinking the root package
owns these concepts. The `state/doc.go` already says "External callers should
import this package directly instead of relying on compatibility aliases."

**Proposal:** Gradually migrate callers to import sub-packages directly and
remove facade files. Add a deprecation notice in each facade file's doc comment
as a transition signal.

---

### 13. Projection-side `ClearRestTemporaryArmor` has hardcoded subclass field resets

**Category:** correctness risk
**File:** `internal/services/game/domain/systems/daggerheart/internal/projection/character_state.go:230-297`

The `ClearRestTemporaryArmor` function contains a 30-line block that manually resets
individual subclass state fields (e.g., `ElementalistActionBonus`,
`TranscendenceActive`, `BattleRitualUsedThisLongRest`). Adding a new subclass
feature with rest-scoped state requires updating this function -- and there is no
compile-time or test-time check to catch a forgotten field.

The same pattern appears in the `ApplyStatePatch` function (lines 131-153) which
has a zero-value check spanning 20 boolean/int fields.

**Proposal:** Define a `Reset()` method on `DaggerheartSubclassState` that
encapsulates the reset logic, or use struct tags / reflection to auto-detect
rest-scoped fields. This centralizes the reset contract and eliminates the risk
of missed fields.

---

### 14. Manifest descriptor wiring is clear but tightly coupled to import paths

**Category:** contributor friction
**File:** `internal/services/game/domain/systems/manifest/manifest.go:39-47`

The `builtInSystems` slice directly imports the `daggerheart` package. A second
system would add a second entry here plus its own import. This is fine for a small
number of systems but:
- The `ProjectionStores` struct has a named `Daggerheart` field (line 16),
  requiring a new named field per system.
- `daggerheartAdapterFromStores` is a system-specific helper at manifest level.

**Proposal:** Consider a `ProjectionStoreProvider` interface pattern where each
system's adapter can extract its own store from a generic provider, rather than
adding named fields to a manifest-level struct. This scales better to 3+ systems.

---

### 15. `contentstore/contracts.go` is 610 lines with 15+ entity types

**Category:** missing best practice
**File:** `internal/services/game/domain/systems/daggerheart/contentstore/contracts.go:1-610`

The content store contract file defines types for classes, subclasses, heritages,
experiences, adversaries, beastforms, companions, loot, damage types, domains,
domain cards, weapons, armor, items, environments, and content strings -- plus
read/write interfaces for each. This is the correct boundary ownership but the
file is very large.

**Proposal:** Split into focused contract files by content domain:
`contracts_character_build.go`, `contracts_encounter.go`, `contracts_equipment.go`,
`contracts_domain.go`, `contracts_localization.go`. This matches the existing
interface grouping (the read/write interfaces are already grouped by domain).

---

### 16. Module interface lacks a lifecycle hook for session-end cleanup

**Category:** missing best practice
**File:** `internal/services/game/domain/module/registry.go:118-131`

The `module.Module` interface has optional extension points for `CharacterReady`
and `SessionStartBootstrap`, but no `SessionEndCleanup` or `RestClearTriggers`
hook. Daggerheart works around this by embedding rest-clear logic inside the
rest decider and projection adapter (findings #13). A second system with
session-scoped state would need to build the same workaround.

**Proposal:** Consider adding an optional `SessionEndHook` interface to the module
contract, similar to `SessionStartBootstrapper`. This would let systems declare
cleanup behavior declaratively rather than embedding it in individual command
handlers.

---

### 17. The `domain/` sub-package name collides with Go convention expectations

**Category:** contributor friction
**File:** `internal/services/game/domain/systems/daggerheart/domain/doc.go`

Having a `domain/` sub-package inside `domain/systems/daggerheart/` creates an
unusual import path: `domain/systems/daggerheart/domain`. This reads as redundant
("domain inside domain") and could confuse contributors scanning import paths.

The package contains duality dice logic (outcome evaluation, probability, action
rolls) which is specific enough to have a more descriptive name.

**Proposal:** Rename to `duality/` or `dice/` to better describe its content
and avoid the redundant `domain/domain` path segment.

---

### 18. What a "getting started" guide for a second system would look like

**Category:** contributor friction (documentation gap)
**Evidence:** `docs/architecture/systems/game-systems.md` provides architecture
but not a step-by-step implementation checklist.

Based on this review, a second system author would need to:

1. Create 8-12 packages under `domain/systems/<name>/`:
   module.go, state/, payload/, profile/, mechanics/, rules/,
   contentstore/, projectionstore/, internal/decider/, internal/folder/,
   internal/adapter/, internal/validator/
2. Define command types in `domain/commandids/` with the `sys.<name>.*` prefix
3. Define event types in `payload/event_types.go`
4. Implement `module.Module` (7 methods)
5. Implement `Decider` with handler map + `DeciderHandledCommands()`
6. Implement `Folder` (event -> snapshot state fold) using `module.FoldRouter`
7. Implement `Adapter` (event -> projection store) using `module.AdapterRouter`
8. Implement `StateFactory` for initial character and snapshot state
9. Define `projectionstore.Store` interface with system-specific methods
10. Define `contentstore` interfaces for catalog content
11. Create a `RegistrySystem` implementing `systems.GameSystem`
12. Add a `SystemDescriptor` entry in `manifest/manifest.go`
13. Add named field to `manifest.ProjectionStores`
14. Implement SQLite storage for projection and content store interfaces
15. Wire gRPC transport handlers

**Proposal:** Create `docs/guides/adding-a-game-system.md` with this checklist,
code snippets from Daggerheart as examples, and cross-references to the
architecture doc. This is the highest-leverage improvement for second-system
onboarding.

---

## Metrics

| Metric | Value |
|--------|-------|
| Total Go files | 199 |
| Test files | 68 |
| Sub-packages | 19 |
| Command types | 35 |
| Event types | 34 |
| Module interface methods | 7 (+ 2 optional extension points) |
| Root facade files | 4 (character_state, death, downtime, rest) |
| Export-alias files | 2 (decider/exports.go, folder/exports.go) |
| Mechanics manifest entries | 42 |
| Lines in contentstore/contracts.go | 610 |
| Lines in projectionstore/contracts.go | 363 |
| Lines in state/character_profile.go | 672 |
