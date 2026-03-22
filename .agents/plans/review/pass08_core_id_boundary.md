# Pass 8: Core ID Package Boundary Leak

## Summary

The `domain/ids` package is documented as a "leaf package with no internal
imports, safe to reference from any domain or infrastructure package" (ids.go:6-8).
It contains 8 core entity ID types (CampaignID, ParticipantID, CharacterID,
SessionID, SceneID, InviteID, UserID, GateID) plus **3 Daggerheart-specific ID
types** (AdversaryID, EnvironmentEntityID, CountdownID) declared at lines 51-64.

Every consumer of these 3 IDs is exclusively within Daggerheart-scoped packages
(domain/systems/daggerheart/\*, api/grpc/systems/daggerheart/\*) with one
exception: `readinesstransport/state.go` uses them to construct a Daggerheart
snapshot inside an already Daggerheart-conditional code path. The core domain
packages (`domain/event`, `domain/command`, `domain/aggregate`) never reference
these IDs.

The boundary violation is real but **low-severity and low-risk**: the IDs are
string newtypes with zero logic, the import graph impact is nil (ids is already
a leaf), and the `domain-language.md` doc does not acknowledge these as
system-specific. Moving them would improve conceptual clarity but carries
non-trivial migration cost for modest architectural gain.

---

## Findings

### Finding 1: Daggerheart-specific IDs in core ids package

**Category:** anti-pattern (boundary violation)
**Severity:** low
**File:** `internal/services/game/domain/ids/ids.go:51-64`

Three ID types carry explicit "Daggerheart" in their doc comments:

```
// AdversaryID identifies a Daggerheart adversary.     (line 51)
// EnvironmentEntityID identifies an instantiated Daggerheart environment. (line 56)
// CountdownID identifies a Daggerheart countdown.     (line 61)
```

These are game-system-specific concepts placed in a package whose stated purpose
is core entity identifiers. The 8 preceding IDs (CampaignID through GateID) are
all system-agnostic campaign model types that appear in core event/command/aggregate
packages. The 3 Daggerheart IDs are not used by any core domain package.

### Finding 2: All typed usages are within Daggerheart-scoped code

**Category:** missing best practice (misplaced types)
**Severity:** low

Grep for `ids.AdversaryID`, `ids.EnvironmentEntityID`, and `ids.CountdownID`
reveals exactly 40 files that reference these typed IDs. All fall into two
categories:

1. **Daggerheart domain packages** (38 files): `domain/systems/daggerheart/**`,
   `api/grpc/systems/daggerheart/**`
2. **Readiness transport** (1 file):
   `api/grpc/game/campaigntransport/readinesstransport/state.go:126-127` -- uses
   them only inside a `if systemID == SystemIDDaggerheart` branch to construct a
   `daggerheartstate.SnapshotState`.
3. **Normalize package** (1 file):
   `domain/normalize/normalize.go:11` -- mentions `ids.AdversaryID` only in a
   doc comment illustrating the generic `ID[T ~string]` function.

No core domain package (`domain/event`, `domain/command`, `domain/aggregate`,
`domain/campaign`, `domain/participant`, `domain/character`, `domain/session`,
`domain/scene`) ever imports or uses these 3 IDs.

### Finding 3: Projection store contracts use plain strings, not typed IDs

**Category:** missing best practice (inconsistency)
**Severity:** informational

The `projectionstore` package (`domain/systems/daggerheart/projectionstore/contracts.go`)
stores all entity IDs as `string` fields (e.g., `AdversaryID string` at line 293,
`CountdownID string` at line 265, `EnvironmentEntityID string` at line 321).
Similarly, the `Store` interface methods accept plain `string` parameters for
these IDs (lines 354, 349, 358).

This means the typed IDs are used only in the domain state/payload/decider layer,
not in the storage contract layer. The type safety benefit is limited to the
domain fold/decide cycle -- the storage boundary already works with untyped
strings.

### Finding 4: domain-language.md does not acknowledge system-specific IDs

**Category:** missing best practice (documentation gap)
**Severity:** informational
**File:** `docs/architecture/foundations/domain-language.md`

The domain language document defines campaign model terms, event-sourcing terms,
session governance terms, and interaction terms. It does not mention adversary,
countdown, or environment entity as domain terms, nor does it acknowledge that
`domain/ids` contains system-specific types. This is consistent with the finding
that these are Daggerheart-specific concepts that do not belong in the core
vocabulary.

### Finding 5: No other system-specific concepts leak into core domain packages

**Category:** positive finding
**Severity:** n/a

Searches for "daggerheart" (case-insensitive) across `domain/command/`,
`domain/event/`, and `domain/aggregate/` found:

- `domain/command/decision.go:15` -- a doc comment example ("e.g.
  GM_FEAR_OUT_OF_RANGE for daggerheart") explaining naming conventions. This is
  appropriate documentation, not a boundary violation.
- `domain/event/registry.go:58` -- a doc comment example ("sys.daggerheart.damage_applied").
  Same: appropriate documentation.
- `domain/event/` and `domain/aggregate/` test files use "daggerheart" only as
  literal string values in test payloads, not as typed dependencies.

The boundary leak is isolated to the 3 ID type declarations in `ids.go`.

### Finding 6: readinesstransport is the only cross-boundary consumer

**Category:** anti-pattern (mild coupling)
**Severity:** low
**File:** `internal/services/game/api/grpc/game/campaigntransport/readinesstransport/state.go:121-128`

This file constructs a `daggerheartstate.SnapshotState` and uses `ids.AdversaryID`
and `ids.CountdownID` as map key types. The usage is already inside a
Daggerheart-conditional branch and imports the Daggerheart state package
directly. The IDs' location in `domain/ids` versus a hypothetical
`daggerheart/ids` package would not change this file's import set since it
already depends on `daggerheartstate`.

---

## Refactoring Proposals

### Option A: Move IDs to a Daggerheart-local ids package (recommended)

Create `domain/systems/daggerheart/ids/ids.go` with:

```go
package ids

type AdversaryID string
func (id AdversaryID) String() string { return string(id) }

type EnvironmentEntityID string
func (id EnvironmentEntityID) String() string { return string(id) }

type CountdownID string
func (id CountdownID) String() string { return string(id) }
```

**Migration path:**

1. Create the new package.
2. Add type aliases in `domain/ids/ids.go` pointing to the new package
   (temporary compatibility shim).
3. Update all 40 consuming files to import from the new package. Since these are
   all within `daggerheart/**` or files that already import daggerheart packages,
   no new cross-boundary imports are introduced.
4. Remove the aliases from `domain/ids/ids.go`.
5. Update `domain/normalize/normalize.go` doc comment.

**Cost:** ~40 files to update (mechanical find-and-replace). No logic changes.
No test changes beyond import paths. The `readinesstransport/state.go` file
already imports daggerheart packages, so adding one more import is not a new
coupling.

**Risk:** Low. These are string newtypes with no methods beyond `String()`. The
generic `normalize.ID[T ~string]` function works with any string-based type
regardless of package.

### Option B: Keep as-is, add doc comment boundary marker

If the migration cost is not justified now, add a section comment in `ids.go`
to make the boundary intent explicit:

```go
// --- System-specific IDs ---
//
// The following IDs are specific to the Daggerheart game system.
// They live here for convenience as leaf-package newtypes, but
// are not core campaign model concepts. Consider moving to
// domain/systems/daggerheart/ids if a second game system is added.
```

**Cost:** Trivial. Documents the known violation for future contributors.

### Option C: Extend typed IDs into projection store contracts

Independently of where the IDs live, the `projectionstore` contracts could use
typed IDs instead of plain strings for `AdversaryID`, `CountdownID`, and
`EnvironmentEntityID` fields and method parameters. This would extend compile-time
safety to the storage boundary.

**Cost:** Moderate. Requires updating `projectionstore/contracts.go` struct
fields and interface signatures, all store implementations
(`daggerheartprojection/store_*.go`), all fake stores, and sqlc-generated code
mappings.

**Assessment:** This is orthogonal to the boundary question and lower priority.

---

## Verdict

The boundary violation is confirmed but low-impact. **Option A** (move to
`daggerheart/ids`) is the architecturally correct fix and should be done when a
second game system is on the roadmap or during a broader system-boundary cleanup.
**Option B** (doc comment) is a reasonable interim step. **Option C** (typed
store contracts) is independent and lower priority.
