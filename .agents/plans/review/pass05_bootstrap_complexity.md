# Pass 5: Registration, Bootstrap, and Startup Complexity

## Summary

The game server has a well-structured 8-phase startup with clear phase boundaries,
typed errors, and injectable seams for testing. The `CoreDomain` struct and manifest
`SystemDescriptor` successfully centralize registration so adding a new aggregate or
system is not a 5-location edit. However, several patterns create contributor
friction and carry mild correctness risk:

1. **Identical contract struct types** are independently redeclared in 4 packages with
   different names, preventing code sharing and creating a naming inconsistency that
   confuses newcomers.
2. **`CoreDomains()` is a hardcoded function-scoped list** -- forgetting to add a new
   domain silently compiles but fails validation at startup, whereas a registry-based
   or init-time approach could surface mistakes earlier.
3. **`SystemDescriptor` duplicates `Module.ID()`/`Module.Version()`** without asserting
   they agree at compile time, relying on a runtime parity check that only runs on
   startup.
4. **Several heavy startup-time validations use `reflect.DeepEqual`** and instantiate
   aggregate objects to verify invariants that could be enforced structurally.
5. **Validation functions hard-call `CoreDomains()` as a global** inside validators
   rather than receiving domains as an explicit parameter, creating hidden coupling.

Despite these issues, the system is admirably defensive: 12+ startup-time validators,
named pipeline steps with clear error wrapping, and rollback-on-failure are all
strong patterns. The findings below focus on the remaining friction and risk.

---

## Findings

### 1. Duplicated contract struct types across aggregates

**Category:** contributor friction, anti-pattern

**Files:**
- `internal/services/game/domain/campaign/registry.go:12-19` (`commandContract`, `eventProjectionContract`)
- `internal/services/game/domain/character/registry.go:12-19` (`commandContract`, `eventProjectionContract`)
- `internal/services/game/domain/session/registry_support.go:8-16` (`commandContract`, `eventProjectionContract`)
- `internal/services/game/domain/scene/registry_support.go:8-16` (`commandContract`, `eventProjectionContract`)
- `internal/services/game/domain/participant/registry.go:11-19` (`commandRegistration`, `eventProjectionRegistration`)

**Detail:** Five packages independently declare structurally identical types:
```go
type commandContract struct {
    definition command.Definition
}
type eventProjectionContract struct {
    definition event.Definition
    emittable  bool
    projection bool
}
```
Participant uses different names (`commandRegistration`, `eventProjectionRegistration`)
with the exact same shape. Campaign and character use `commandContract`/`eventProjectionContract`,
session and scene use the same names but define them in `_support.go`.

**Risk:** A contributor adding a new field (e.g., `foldOnly bool`) must find and update
all five redeclarations manually. The naming inconsistency in `participant` makes grep-based
discovery unreliable.

**Proposal:** Extract a single shared type into `domain/engine/` or a dedicated
`domain/registry/` package:
```go
type CommandContract struct {
    Definition command.Definition
}
type EventContract struct {
    Definition event.Definition
    Emittable  bool
    Projection bool
}
```
All aggregate registry files import and use the shared type.

---

### 2. `CoreDomains()` is a hardcoded list -- fragile on addition

**Category:** correctness risk, contributor friction

**File:** `internal/services/game/domain/engine/core_domain.go:35-98`

**Detail:** `CoreDomains()` returns a literal `[]CoreDomain{...}` with 6 entries
(campaign, action, session, participant, character, scene). Adding a 7th domain
requires editing this one function. If a contributor creates the package, wires its
registry.go functions, but forgets to add the entry here, the code compiles
successfully. The failure only surfaces at startup when validation checks detect the
gap.

The comment on line 33-34 acknowledges this: "adding a new domain is a single append
rather than editing 5+ locations." While this is better than the alternative, it is
still a manual step that requires insider knowledge.

**Risk:** Silent compilation with runtime failure. The startup validators do catch
the gap, but only when the server actually boots -- not during CI `go build`.

**Proposal:** Consider an init-time registration pattern where each domain package
calls `engine.RegisterCoreDomain(...)` in an `init()` function or explicit
constructor, making the list self-assembling. Alternatively, add a go:generate-based
test that verifies `CoreDomains()` covers all packages under `domain/` that export
`RegisterCommands`.

---

### 3. `SystemDescriptor.ID`/`Version` duplicates `Module.ID()`/`Module.Version()`

**Category:** anti-pattern, correctness risk

**Files:**
- `internal/services/game/domain/systems/manifest/manifest.go:28-37` (SystemDescriptor)
- `internal/services/game/domain/systems/manifest/manifest.go:39-47` (builtInSystems)

**Detail:** `SystemDescriptor` has explicit `ID` and `Version` string fields, but the
`BuildModule` func it carries returns a `Module` that also has `ID()` and `Version()`
methods. The manifest hardcodes `daggerheart.SystemID` and `daggerheart.SystemVersion`
on the descriptor (line 41-42) and separately the built module also returns those same
values. If someone changes the module constant but not the descriptor (or vice versa),
the values drift.

The runtime parity check in `app/system_registration.go:41-94`
(`validateSystemRegistrationParity`) catches module-vs-metadata drift, but the
descriptor-vs-module drift is not explicitly validated.

**Proposal:** Remove `ID` and `Version` from `SystemDescriptor`. Instead, derive them
from `BuildModule().ID()` and `BuildModule().Version()` at manifest construction time.
The descriptor becomes:
```go
type SystemDescriptor struct {
    BuildModule         func() module.Module
    BuildMetadataSystem func() bridge.GameSystem
    BuildAdapter        func(ProjectionStores) bridge.Adapter
}
```
`Modules()`, `MetadataSystems()`, and `AdapterRegistry()` already call `BuildModule()`
and can extract ID/Version from the returned module.

---

### 4. Per-aggregate registration boilerplate (5 functions x 6 domains = 30 functions)

**Category:** contributor friction, anti-pattern

**Files:** All 6 aggregate registry files under `domain/`:
- `campaign/registry.go` -- `RegisterCommands`, `RegisterEvents`, `EmittableEventTypes`, `DeciderHandledCommands`, `ProjectionHandledTypes` (+ `FoldHandledTypes`, `RejectionCodes` in fold.go/decider.go)
- Same 5+ functions in `session/`, `participant/`, `character/`, `scene/`, `action/`

**Detail:** Every aggregate exports the same 5-7 package-level functions with
identical structure:
1. `RegisterCommands(*command.Registry) error` -- iterates contracts, calls `registry.Register`
2. `RegisterEvents(*event.Registry) error` -- same pattern
3. `EmittableEventTypes() []event.Type` -- filters contracts by `emittable`
4. `DeciderHandledCommands() []command.Type` -- extracts types from command contracts
5. `ProjectionHandledTypes() []event.Type` -- filters by `projection`

The iteration logic is copy-pasted across all 6 packages. Each function body is 5-10
lines of the same structure.

**Risk:** Mostly boilerplate friction. A contributor adding a new aggregate must copy
one of the existing registry files and adapt it, but the mechanical nature of the code
means subtle mistakes (forgetting a filter) are possible.

**Proposal:** Define a shared `DomainRegistrar` type in `engine/` or `domain/registry/`:
```go
type DomainRegistrar struct {
    Commands []CommandContract
    Events   []EventContract
}
func (r DomainRegistrar) RegisterCommands(reg *command.Registry) error { ... }
func (r DomainRegistrar) RegisterEvents(reg *event.Registry) error { ... }
func (r DomainRegistrar) EmittableEventTypes() []event.Type { ... }
// etc.
```
Each aggregate declares `var registrar = engine.DomainRegistrar{...}` and its
exported functions become one-liners delegating to the shared implementation.

---

### 5. Validation functions hard-call `CoreDomains()` as a global

**Category:** anti-pattern, missing best practice

**Files:**
- `internal/services/game/domain/engine/registries_validation_core.go:15` -- `validateCoreEmittableEventTypes` calls `CoreDomains()`
- `internal/services/game/domain/engine/registries_validation_core.go:41` -- `ValidateFoldCoverage` calls `CoreDomains()`
- `internal/services/game/domain/engine/registries_validation_core.go:101` -- `ValidateEntityKeyedAddressing` calls `CoreDomains()`
- `internal/services/game/domain/engine/registries_validation_core.go:154` -- `ValidateAliasFoldCoverage` calls `CoreDomains()`
- `internal/services/game/domain/engine/registries_validation_core.go:186` -- `ValidateCoreRejectionCodeUniqueness` calls `CoreDomains()`
- `internal/services/game/domain/engine/registries_validation_core.go:217` -- `ValidateCoreDeciderCommandCoverage` calls `CoreDomains()`
- `internal/services/game/domain/engine/registries_validation_aggregate.go:26` -- `ValidateAggregateFoldDispatch` calls `CoreDomains()`
- `internal/services/game/domain/engine/registries_validation_projection.go:37` -- `ValidateProjectionRegistries` calls `CoreDomains()`

**Detail:** Eight validation functions call `CoreDomains()` directly instead of
receiving the domain list as a parameter. This makes them untestable with custom
domain sets and creates a hidden global dependency. The contract validator
(`registries_contract_validation.go`) correctly receives `domains` as a constructor
parameter and passes them through, but individual validators bypass this by calling
the global directly.

**Risk:** Tests cannot inject synthetic domains to exercise edge cases. Every
validation function also allocates a fresh `CoreDomains()` slice on each call.

**Proposal:** Thread `[]CoreDomain` through as an explicit parameter to each
validation function. The `registryContractValidator` already demonstrates the right
pattern -- it receives domains in `newRegistryContractValidator` and passes them to
`collectFoldHandledTypes` and `collectProjectionHandledTypes`. The remaining
validators should follow the same approach.

---

### 6. `reflect.DeepEqual` in startup validation (`ValidateStateFactoryDeterminism`)

**Category:** missing best practice

**File:** `internal/services/game/domain/engine/registries_validation_system.go:229-267`

**Detail:** `ValidateStateFactoryDeterminism` calls `NewSnapshotState` and
`NewCharacterState` twice each with the same inputs, then compares results with
`reflect.DeepEqual`. This catches non-deterministic implementations, but
`reflect.DeepEqual` is heavy-handed for startup validation -- it traverses all
fields including unexported ones, private mutexes, function pointers, etc.

**Risk:** A state factory that embeds a `sync.Mutex` or function-valued field
would fail `DeepEqual` even if the domain state is logically deterministic. The
check also prevents using state factories that return interface values with
different underlying pointers.

**Proposal:** Consider a lighter determinism contract: require state factories to
implement a `DeterminismKey() string` method, or compare only serialized JSON
output. Alternatively, document that state factory return values must be
`reflect.DeepEqual`-safe and keep the current approach with that explicit
constraint.

---

### 7. Duplicate registry infrastructure (`module.Registry` vs `systems.MetadataRegistry`)

**Category:** contributor friction, anti-pattern

**Files:**
- `internal/services/game/domain/module/registry.go:134-298` -- `Registry` with `Key{ID, Version}`
- `internal/services/game/domain/systems/registry_bridge.go:229-336` -- `MetadataRegistry` with `SystemKey{ID, Version}`

**Detail:** Two separate registry types manage system modules with nearly identical
structure:
- `module.Registry` -- maps `Key{ID, Version}` to `Module`, tracks defaults
- `systems.MetadataRegistry` -- maps `SystemKey{ID, Version}` to `GameSystem`, tracks defaults

Both have `Register`, `Get`/`GetVersion`, `DefaultVersion`, `List` methods with the
same semantics. `AdapterRegistry` (adapter_registry.go:26-145) is a third registry
with the same key structure (`systemKey{ID, Version}`) and similar operations.

The manifest `SystemDescriptor` ties all three together, but the three registries are
independently coded and tested.

**Risk:** If registration semantics change (e.g., version normalization rules), the
change must be applied three times. Bug fixes in one registry may not be propagated
to the others.

**Proposal:** Extract a generic `VersionedRegistry[T any]` with the common
registration, lookup, and default-version semantics. Each specialized registry
becomes a thin wrapper:
```go
type Registry = VersionedRegistry[Module]
type MetadataRegistry = VersionedRegistry[GameSystem]
type AdapterRegistry = VersionedRegistry[Adapter]
```

---

### 8. Bootstrap dependency wiring is manual cleanup-on-error

**Category:** missing best practice

**File:** `internal/services/game/app/bootstrap.go:69-151` (`managedConnDependencyDialer.Dial`)

**Detail:** The `Dial` method creates 4 managed connections sequentially. On each
failure, it manually closes all previously opened connections:
```go
if err != nil {
    authMc.Close()
    socialMc.Close()
    aiMc.Close()
    return ...
}
```
The cleanup logic grows linearly with each new connection. If a 5th connection is
added, the developer must remember to close all 4 prior connections in the new error
path.

**Risk:** Missing a cleanup call causes resource leaks on startup failure. The
pattern is also fragile to reordering.

**Proposal:** Use the `startupRollback` struct that already exists in the bootstrap
package. Pass it into `Dial` or restructure the method to accumulate cleanup functions:
```go
rollback.add(func() { authMc.Close() })
// ... next dial ...
rollback.add(func() { socialMc.Close() })
```
The caller's existing `rollback.cleanup()` handles reverse-order teardown on failure.

---

### 9. `app/bootstrap_registration_assembly.go` manually threads 40+ store references

**Category:** contributor friction

**File:** `internal/services/game/app/bootstrap_registration_assembly.go:33-90`

**Detail:** `buildRegistrationAssemblies` manually copies ~40 store references from
`configuredDomainState` and `registrationAssemblySources` into 4 separate
`*RegistrationDeps` structs. Many stores are duplicated across structs (e.g.,
`campaignStore` appears in campaign, session, and infrastructure deps; `participantStore`
appears in all four).

**Risk:** Adding a new store requires updating multiple dep structs and the assembly
function. Forgetting one site causes a nil-store panic at runtime, not a compile error,
because the dep structs use concrete interface types that default to nil.

**Proposal:** Consider a shared `ProjectionStoreBundle` struct that all dep structs
embed or reference. This reduces the assembly function to setting 4 bundle references
instead of 40 individual fields:
```go
type campaignRegistrationDeps struct {
    stores ProjectionStoreBundle
    // ... campaign-specific fields ...
}
```

---

### 10. `CoreDomain` struct uses function fields instead of an interface

**Category:** anti-pattern (mild)

**File:** `internal/services/game/domain/engine/core_domain.go:18-27`

**Detail:** `CoreDomain` is a struct with 7 function-typed fields
(`RegisterCommands`, `RegisterEvents`, `EmittableEventTypes`, `FoldHandledTypes`,
`DeciderHandledCommands`, `ProjectionHandledTypes`, `RejectionCodes`). Each aggregate
package must export matching package-level functions and the `CoreDomains()` function
manually wires them.

The `Module` interface (module/registry.go:119-131) demonstrates the interface-based
alternative: each system module implements the interface directly, and registration
is `registry.Register(module)`.

**Risk:** A `nil` function field on `CoreDomain` is a silent runtime panic rather
than a compile error. The comment says "the compiler catches the rest" but the
compiler does not enforce that all function fields are non-nil on a struct literal.

**Proposal:** Define a `CoreDomainRegistrar` interface with the 7 methods. Each
aggregate package exports a type satisfying the interface. `CoreDomains()` returns
`[]CoreDomainRegistrar`. The compiler then enforces method presence.

---

### 11. Session/scene `appendSessionCommandContracts`/`appendSceneCommandContracts` are identical

**Category:** contributor friction, anti-pattern

**Files:**
- `internal/services/game/domain/session/registry_support.go:36-58`
- `internal/services/game/domain/scene/registry_support.go:36-58`

**Detail:** Both packages define `appendXxxCommandContracts` and
`appendXxxEventContracts` helper functions that are structurally identical -- they
concatenate variadic `[]commandContract` slices. The only difference is the
function name prefix (`session` vs `scene`).

Combined with finding #1 (the types themselves are identical), these helpers are
pure boilerplate duplication.

**Proposal:** If the contract types are unified per finding #1, these helpers become
a single shared `AppendContracts` function.

---

### 12. `ValidateAggregateFoldDispatch` instantiates `aggregate.Folder{}` at validation time

**Category:** missing best practice

**File:** `internal/services/game/domain/engine/registries_validation_aggregate.go:19`

**Detail:** The validation function creates a zero-value `aggregate.Folder{}` just
to call `FoldDispatchedTypes()` on it. This works because `FoldDispatchedTypes()`
returns a static slice, but the pattern couples the validator to the concrete
aggregate folder type.

**Risk:** If `aggregate.Folder` gains required constructor arguments (e.g., a registry
dependency), this validator silently produces incorrect results or panics.

**Proposal:** Either:
(a) Make `FoldDispatchedTypes()` a package-level function in `aggregate/` instead of
a method on `Folder`, or
(b) Pass the list of dispatched types as a parameter to the validator.

---

### 13. Phase 8 (Runtime) projection mode resolution depends on env vars parsed in Phase 1

**Category:** missing best practice (minor)

**File:** `internal/services/game/app/bootstrap.go:172-178` (`defaultProjectionRuntimeConfigurer.Configure`)

**Detail:** `resolveProjectionApplyModes` receives the full `serverEnv` struct,
which was loaded in phase 1. The runtime configurer thus has indirect access to all
environment state, not just the projection-related fields. This is a wide dependency
surface for what should be a narrow configuration concern.

**Risk:** A contributor might read additional env fields in the runtime configurer
without realizing the conceptual boundary between phases. The current code is
disciplined, but the signature does not enforce it.

**Proposal:** Replace `serverEnv` with a narrow `ProjectionRuntimeConfig` struct
containing only the relevant fields (`ProjectionApplyOutboxEnabled`,
`ProjectionApplyOutboxShadow`, etc.).

---

### 14. Startup error wrapping is thorough but `startupPhase` constants are not exhaustive-checked

**Category:** missing best practice (minor)

**File:** `internal/services/game/app/startup_errors.go:8-17`

**Detail:** `startupPhase` is a `string` type alias with 8 constants. There is no
compile-time check that every phase is used exactly once in the bootstrap sequence.
A typo in a phase name would still compile.

**Risk:** Very low. The phase names are only used in error messages, so a typo just
degrades diagnostics rather than causing incorrect behavior.

**Proposal:** No code change needed. Consider adding a test that asserts
`NewWithAddr` exercises all 8 phases in order if phase coverage becomes important.

---

### 15. `buildRegistriesPhase` calls `registeredSystemModules()` -- a second global call site

**Category:** contributor friction (minor)

**File:** `internal/services/game/app/bootstrap_sequence.go:25-31`

**Detail:** `buildRegistriesPhase` calls `registeredSystemModules()` to get the
module list. Later, `bootstrapSystemsPhase` (via `defaultSystemsBootstrapper.Bootstrap`)
calls `registeredSystemModules()` again (bootstrap_systems.go:42). This means the
module list is constructed twice from the manifest. If the list were mutable (it is
not today), the two call sites could observe different values.

**Risk:** Negligible with the current immutable manifest. The double call is
wasteful but harmless.

**Proposal:** Build the module list once in `NewWithAddr` and pass it through as
a parameter to both phases.

---

### 16. Campaign event contracts omit explicit `Intent` field

**Category:** correctness risk (mitigated)

**File:** `internal/services/game/domain/campaign/registry.go:105-166`

**Detail:** Campaign event definitions do not set `Intent` explicitly. The event
registry defaults unset `Intent` to `IntentProjectionAndReplay` (event/registry.go:196-198).
All other aggregates with projection events (session, scene) set `Intent` explicitly.
Action events set `IntentReplayOnly` and `IntentAuditOnly` explicitly.

**Risk:** The default behavior is correct for campaign events (they do need projection
and replay). However, relying on the default makes the intent invisible to readers
and inconsistent with the other aggregates that declare it explicitly.

**Proposal:** Add explicit `Intent: event.IntentProjectionAndReplay` to all campaign
event definitions for clarity and consistency with session/scene patterns.

---

### 17. `ValidateProjectionRegistries` called twice during startup

**Category:** contributor friction (minor)

**Files:**
- `internal/services/game/domain/engine/registries_contract_validation.go:74-77` -- called during `ValidateProjection` in registry build phase
- `internal/services/game/app/server_bootstrap.go:54-60` -- called again during `buildProjectionRegistries` in runtime phase

**Detail:** Projection registry validation runs once during the registry build workflow
(`registryContractValidator.ValidateProjection`) and again during the projection
runtime configuration phase (`buildProjectionRegistries`). The second call includes
`adapters` (which is nil in the first call), so it is not fully redundant, but the
core/projection coverage checks run twice.

**Risk:** Startup latency. The validators are fast, so the impact is negligible, but
the duplication is conceptually confusing.

**Proposal:** Remove the first call from the registry build workflow since the second
call in the runtime phase is more complete (includes adapter validation). Or
restructure so the adapter-aware validation is the only call.
