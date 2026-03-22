# Pass 4: Interface Design and Dependency Injection

## Summary

The interface design across the game service domain layer is largely well-structured. The `Module` interface (8 methods) is appropriately sized given it represents a full game-system plugin contract. The `Applier` struct's 19 fields are mitigated by `StoreGroups` and `NewBoundApplier` constructors, though the flat struct remains for compatibility. The three Folder interfaces (`engine.Folder`, `fold.Folder`, `replay.Folder`) are an intentional design choice documented inline, though the proliferation creates cognitive overhead. Optional interfaces via type assertion are well-documented but carry discoverability risk. The `Deps` pattern is consistent where used.

Findings are ordered by severity: correctness risks first, then anti-patterns, then friction items.

---

## Findings

### 1. Three parallel Folder interfaces with subset relationships

**Category:** contributor friction
**Files:**
- `internal/services/game/domain/fold/fold.go:17-20` -- canonical `fold.Folder` (Fold + FoldHandledTypes)
- `internal/services/game/domain/engine/handler.go:88-90` -- `engine.Folder` (Fold only)
- `internal/services/game/domain/replay/replay.go:50-52` -- `replay.Folder` (Fold only)
- `internal/services/game/domain/module/registry.go:42` -- `module.Folder = fold.Folder` (type alias)

**Description:** There are three distinct Folder interfaces across the codebase:

1. `fold.Folder` -- the canonical 2-method interface (Fold + FoldHandledTypes), used by startup validators
2. `engine.Folder` -- a 1-method subset (Fold only), used by Handler
3. `replay.Folder` -- a 1-method subset (Fold only), used by replay pipeline
4. `module.Folder` -- type alias of `fold.Folder`

Both `engine.Folder` and `replay.Folder` are documented as "narrow local interface" for their consumer, and `module.Folder` is an explicit alias. The documentation in both narrow interfaces references `fold.Folder` as canonical.

This follows Go ISP (Interface Segregation Principle) correctly -- consumers define the narrowest interface they need. However, a contributor working with the fold layer for the first time will encounter three interfaces named "Folder" with different method sets, which creates confusion about which one to implement or accept.

**Proposal:** This is currently working as designed and documented. The inline comments adequately explain the relationship. No refactor needed unless a second contributor reports confusion. If friction increases, consider eliminating the local definitions and having engine/replay accept `fold.Folder` directly (since FoldHandledTypes is a read-only method, accepting a wider interface imposes no burden on callers).

---

### 2. Module interface (8 methods) -- appropriate size for the plugin contract

**Category:** not an issue (documented analysis)
**File:** `internal/services/game/domain/module/registry.go:119-131`

**Description:** The `Module` interface has 8 methods:

```
ID() string
Version() string
RegisterCommands(*command.Registry) error
RegisterEvents(*event.Registry) error
EmittableEventTypes() []event.Type
Decider() Decider
Folder() Folder
StateFactory() StateFactory
```

These decompose into three natural groups:

1. **Identity** (2): `ID()`, `Version()` -- required for registry keying
2. **Registration** (3): `RegisterCommands()`, `RegisterEvents()`, `EmittableEventTypes()` -- startup wiring
3. **Runtime** (3): `Decider()`, `Folder()`, `StateFactory()` -- request-time collaborators

All 8 methods are called by the engine bootstrap pipeline or runtime routing. There is exactly one production implementation (`daggerheart.Module`), and the interface represents a complete game-system plugin surface. Splitting it into sub-interfaces (e.g., `Registrar` + `RuntimeModule`) would add abstraction without reducing coupling, since `BuildRegistries` needs all 8 methods from the same instance.

**Proposal:** No change. The interface is cohesive and right-sized for its role.

---

### 3. Optional interfaces via type assertion -- discoverability gap

**Category:** contributor friction
**Files:**
- `internal/services/game/domain/module/registry.go:53-55` -- `CharacterReadinessChecker`
- `internal/services/game/domain/module/registry.go:68-75` -- `SessionStartBootstrapper`
- `internal/services/game/domain/module/registry.go:81-83` -- `CommandTyper`
- `internal/services/game/api/grpc/internal/domainwrite/transport.go:170-172` -- `auditStoreDeps`

**Description:** Four optional interfaces are discovered at runtime via type assertion:

1. `CharacterReadinessChecker` -- checked by `ValidateSystemReadinessCheckerCoverage` (engine startup) and `session_start_workflow.go:93`. Currently **mandatory** per startup validation (registries_validation_system.go:41-62), which contradicts the doc comment saying "Modules that do not implement this interface fall back to core readiness rules only." The startup validator rejects modules that don't implement it.

2. `SessionStartBootstrapper` -- checked by `session_start_workflow.go:119`. Truly optional; modules that skip it contribute no bootstrap events.

3. `CommandTyper` -- checked by `ValidateDeciderCommandCoverage` (registries_validation_system.go:120-121). Required only if the module registers system commands.

4. `auditStoreDeps` -- checked by `setDefaultOnRejection` (transport.go:182). Truly optional; falls back to structured logging.

The first three are co-located in `module/registry.go` alongside `Module`, which helps. The fourth is unexported and local to `domainwrite/transport.go`. A new game-system author must read the startup validators to discover that `CharacterReadinessChecker` is effectively mandatory despite its "optional" doc comment.

**Proposal:**
- (a) Fix the doc comment on `CharacterReadinessChecker` (registry.go:44-52) to state it is **required** by startup validation, or make the validator truly fall back gracefully. The current code at registries_validation_system.go:41-62 makes it mandatory; the doc comment at registry.go:44-52 says it is optional. One of them is wrong.
- (b) Add a "System Module Extension Points" section to the game-system skill or docs listing all four optional interfaces, their discovery points, and whether they are truly optional or effectively mandatory.
- (c) No change needed for `auditStoreDeps` -- it is unexported, local, and correctly scoped.

---

### 4. Applier 19-field struct with StoreGroups partial decomposition

**Category:** anti-pattern (partial migration)
**Files:**
- `internal/services/game/projection/applier.go:17-62` -- flat Applier struct (19 fields)
- `internal/services/game/projection/applier_construction.go:15-64` -- StoreGroups decomposition
- `internal/services/game/projection/applier_construction.go:172-195` -- NewBoundApplier re-flattens

**Description:** The `StoreGroups` type decomposes the Applier's store fields into 6 concern-local groups (CampaignStores, ParticipantStores, CharacterStores, SessionStores, SceneStores, SupportStores). However, `NewBoundApplier` immediately re-flattens them back into the 19-field `Applier` struct:

```go
func NewBoundApplier(config BoundApplierConfig) Applier {
    return Applier{
        Campaign:           config.Stores.Campaign,
        Character:          config.Stores.Character,
        // ... 15 more fields individually assigned
    }
}
```

The Applier struct itself remains flat with 19 individual fields. The `StoreGroups` decomposition only helps at construction time; method bodies on `Applier` still reference `a.Campaign`, `a.Session`, etc. individually.

This is a half-completed migration: `StoreGroups` provides grouped construction, but `Applier` never adopted the groups as embedded fields.

**Proposal:** Complete the migration by embedding `StoreGroups` (or its sub-groups) directly in `Applier`:

```go
type Applier struct {
    Events   *event.Registry
    Stores   StoreGroups
    Adapters *bridge.AdapterRegistry
    Now      func() time.Time
    Auditor  *audit.Emitter
}
```

This would reduce Applier from 19 fields to 5, and method bodies would use `a.Stores.Campaign`, `a.Stores.Session`, etc. The `NewBoundApplier` constructor would simplify to direct assignment. This is a medium-sized refactor touching the core router handlers and watermark code that reference individual store fields.

---

### 5. Handler struct (12 fields) -- well-documented, appropriately sized

**Category:** not an issue (documented analysis)
**File:** `internal/services/game/domain/engine/handler.go:120-133`

**Description:** The Handler struct has 12 fields:

- 4 required: Commands, Events, Journal, Decider
- 2 conditionally required: GateStateLoader, SceneGateStateLoader
- 6 optional: Checkpoints, Snapshots, Gate, StateLoader, Folder, Now

The `NewHandler` constructor validates required fields and conditionally-required fields. Optional fields are nil-safe at call sites. The doc comment (lines 100-119) explicitly categorizes each field as required, conditionally required, or optional.

This is a value-type struct (no pointer receiver) intentionally: "Handler struct remains exported for test flexibility where only a subset of fields is needed" (line 137-138). The field count is reasonable for a domain orchestrator.

**Proposal:** No change. The field count is justified by the 7-step execution pipeline. Each field maps to a distinct step (validate, gate, load, decide, validate-events, append, fold + checkpoint/snapshot). The doc comments are exemplary.

---

### 6. Deps pattern consistency in domainwrite transport

**Category:** missing best practice
**File:** `internal/services/game/api/grpc/internal/domainwrite/transport.go:22-26`

**Description:** The `Deps` interface has 2 methods:

```go
type Deps interface {
    DomainExecutor() Executor
    DomainWriteRuntime() *Runtime
}
```

`WritePath` satisfies `Deps` and adds an `AuditEventStore()` method via the optional `auditStoreDeps` interface. This is consumed by `TransportExecuteAndApply` and `TransportExecuteWithoutInlineApply`.

The pattern is consistent across all usages found (game stores, daggerheart stores, snapshot transport). All transport packages use `domainwrite.WritePath` as the concrete type. The `Deps` interface exists primarily to allow `setDefaultOnRejection` to probe for `auditStoreDeps` via the narrower interface.

However, the `Deps` interface returns a concrete `*Runtime` pointer from `DomainWriteRuntime()`, which breaks the interface abstraction. Both callers immediately access `Runtime` methods, so the return type could be an interface. This is minor -- `Runtime` is an internal package type with no external implementations.

**Proposal:** No change needed. The concrete `*Runtime` return is acceptable because `Runtime` is internal to the `domainwrite` package. The `Deps` interface is small, consistent, and used uniformly.

---

### 7. GameSystem interface (6 methods) vs Module interface (8 methods) -- parallel but separate

**Category:** contributor friction
**File:** `internal/services/game/domain/systems/registry_bridge.go:83-106`

**Description:** `systems.GameSystem` and `module.Module` are two separate interfaces that a game system author must implement for a complete system:

- `module.Module` (8 methods) -- write-path: command/event registration, decider, folder, state factory
- `systems.GameSystem` (6 methods) -- API bridge: identity, name, metadata, state handler factory, outcome applier

Both have `ID()` and `Version()` methods but with different return types (`string` vs `SystemID`). A Daggerheart system has two separate struct types implementing each:
- `daggerheart.Module` implements `module.Module`
- `daggerheart.RegistrySystem` implements `systems.GameSystem`

This separation is documented in both `module/doc.go` and `systems/doc.go` as intentional: "module.Registry -- write-path module routing" vs "systems.MetadataRegistry -- API metadata registry." However, for a second-system author, the requirement to implement two separate interfaces in two separate packages is non-obvious.

**Proposal:** Document the dual-interface requirement in the `game-system` skill and/or in `docs/architecture/`. The separation is architecturally correct (write-path vs read/API-path), but the onboarding path needs explicit guidance. The `module/testkit/conformance.go` already validates `Module` + `Adapter` together, which helps; extending it to cover `GameSystem` registration would complete the coverage.

---

### 8. Adapter interface (5 methods) mirrors Module structure -- no issues

**Category:** not an issue (documented analysis)
**File:** `internal/services/game/domain/systems/adapter_registry.go:14-23`

**Description:** The `Adapter` interface has 5 methods:

```
ID() string
Version() string
Apply(context.Context, event.Event) error
Snapshot(context.Context, string) (any, error)
HandledTypes() []event.Type
```

This is the projection-side counterpart of `module.Module`'s fold path. Only one production implementation exists (Daggerheart adapter). The interface is cohesive and right-sized.

**Proposal:** No change.

---

### 9. StateHandlerFactory vs StateFactory -- naming collision risk

**Category:** contributor friction
**Files:**
- `internal/services/game/domain/module/registry.go:113-116` -- `module.StateFactory` (returns `any`)
- `internal/services/game/domain/systems/registry_bridge.go:191-197` -- `systems.StateHandlerFactory` (returns typed handlers)

**Description:** Two interfaces serve similar purposes with different return types:

1. `module.StateFactory` -- write-path: `NewCharacterState(campaignID, characterID, kind) (any, error)` and `NewSnapshotState(campaignID) (any, error)`
2. `systems.StateHandlerFactory` -- API bridge: `NewCharacterState(campaignID, characterID, kind) (CharacterStateHandler, error)` and `NewSnapshotState(campaignID) (SnapshotStateHandler, error)`

The doc comments on `StateHandlerFactory` (registry_bridge.go:181-190) explicitly explain the naming difference and relationship to `module.StateFactory`. The `GameSystem.StateHandlerFactory()` accessor is named to avoid ambiguity.

**Proposal:** The documentation is adequate. No change needed, but the game-system skill should reference both factories and clarify when each is used (fold-path vs API-bridge-path).

---

### 10. Speculative interfaces with single implementations

**Category:** anti-pattern (minor)
**Files:**
- `internal/services/game/domain/systems/registry_bridge.go:126-129` -- `Healable` interface (1 use: Daggerheart)
- `internal/services/game/domain/systems/registry_bridge.go:131-134` -- `Damageable` interface (1 use: Daggerheart)
- `internal/services/game/domain/systems/registry_bridge.go:138-156` -- `ResourceHolder` interface (1 use: Daggerheart)
- `internal/services/game/domain/systems/registry_bridge.go:221-225` -- `OutcomeApplier` interface (1 use: Daggerheart)

**Description:** The `Healable`, `Damageable`, `ResourceHolder`, and `OutcomeApplier` interfaces in `domain/systems` each have exactly one implementation (Daggerheart). Per CLAUDE.md: "Define interfaces at consumption points; avoid speculative interfaces."

These are embedded in `CharacterStateHandler` and `SnapshotStateHandler`, which are returned by `StateHandlerFactory`. The decomposition into `Healable + Damageable + ResourceHolder` is speculative -- a second game system might not decompose character state along these exact lines.

However, these interfaces live in the `systems` package which is explicitly designed as the game-system extension point. Premature abstraction is a known risk here, but the cost is low (small interfaces, no logic) and the benefit is clear documentation of the expected capability surface for game systems.

**Proposal:** Acceptable as-is given the extension-point role. Flag for review when a second game system is added -- if the second system doesn't cleanly fit `Healable/Damageable/ResourceHolder`, the interfaces should be removed in favor of system-specific APIs.

---

### 11. AdapterRegistry and MetadataRegistry duplicate key/registration patterns

**Category:** anti-pattern (code duplication)
**Files:**
- `internal/services/game/domain/systems/adapter_registry.go:26-36` -- `AdapterRegistry` with `systemKey{ID, Version}`
- `internal/services/game/domain/systems/registry_bridge.go:229-239` -- `MetadataRegistry` with `SystemKey{ID, Version}`
- `internal/services/game/domain/module/registry.go:134-138` -- `module.Registry` with `Key{ID, Version}`

**Description:** Three registries use nearly identical patterns:

| Registry | Key type | mutex | defaults map | Register/Get/List |
|---|---|---|---|---|
| `module.Registry` | `Key{ID, Version string}` | `sync.RWMutex` | `map[string]string` | yes |
| `AdapterRegistry` | `systemKey{ID, Version string}` | `sync.RWMutex` | `map[string]string` | yes |
| `MetadataRegistry` | `SystemKey{ID SystemID, Version string}` | `sync.RWMutex` | `map[SystemID]string` | yes |

Each has the same registration flow: check nil, validate ID/version, check duplicates, store in map, set default version. The code is nearly identical across all three.

**Proposal:** Extract a generic `TypedRegistry[K comparable, V any]` that handles the common key/default/mutex pattern. Each registry would embed it and add domain-specific validation. This is a medium-priority cleanup -- the duplication is stable and unlikely to diverge, but it means three places to update if the pattern changes (e.g., adding version ordering or deprecation).

---

### 12. CoreDomain struct uses function fields instead of an interface

**Category:** not an issue (documented analysis)
**File:** `internal/services/game/domain/engine/core_domain.go:18-27`

**Description:** `CoreDomain` uses function fields (`RegisterCommands func(*command.Registry) error`, etc.) rather than defining an interface that each domain package implements. This is intentional: it allows each domain package to export standalone functions (`campaign.RegisterCommands`, `session.RegisterEvents`, etc.) without requiring a wrapper struct.

The function-field approach eliminates the need for domain packages to define a "module" or "registration" type just to satisfy an interface. Since core domains are a fixed, small set (6 entries), the flexibility benefit outweighs the interface formality cost.

**Proposal:** No change. The function-field approach is appropriate for the fixed set of core domains.

---

### 13. registryBootstrap uses historical test seams alongside explicit collaborators

**Category:** anti-pattern (dual path)
**File:** `internal/services/game/domain/engine/registries_contract_validation.go:130-146`

**Description:** Three methods on `registryBootstrap` exist solely as "historical test seams":

- `validateRegistryContracts` (line 132) -- delegates to `registryContractValidator.ValidateWritePath`
- `validateProjectionRegistries` (line 138) -- delegates to `registryContractValidator.ValidateProjection`
- `validatePayloadValidators` (line 144) -- delegates to `registryPayloadValidator.Validate`

Each has a comment like "keeps the historical test seam while delegating..." Similarly in `registries_core_domains.go`:

- `registerCoreDomains` (line 41) -- delegates to `registryCoreDomainRegistrar.Register`
- `validateCoreRegistrations` (line 47) -- delegates to `registryCoreDomainRegistrar.Validate`

These shim methods exist only for test compatibility and add indirection without value. Per CLAUDE.md: "Internal compatibility shims are temporary and must include removal criteria."

**Proposal:** Migrate tests to call the explicit collaborator types directly and delete the shim methods. The shim comments say "keeps the historical test seam" but provide no removal timeline.

---

### 14. NewBoundApplier field-by-field copy from StoreGroups

**Category:** anti-pattern (boilerplate)
**File:** `internal/services/game/projection/applier_construction.go:172-195`

**Description:** `NewBoundApplier` manually copies 15 store fields from `config.Stores.*` sub-groups to the flat `Applier` struct. This is the same finding as #4 from the other direction -- the flat Applier struct forces a field-by-field expansion.

Any addition of a new projection store requires updates in:
1. The flat `Applier` struct
2. The relevant `StoreGroups` sub-type
3. `NewBoundApplier` (the copy)
4. `StoreGroupsFromBundle` (the expansion)

Four locations for one new store field is excessive.

**Proposal:** Same as finding #4 -- embed `StoreGroups` in `Applier` to eliminate the copy step.

---

### 15. Storage contract split -- well-structured Reader/Store separation

**Category:** not an issue (documented analysis)
**Files:**
- `internal/services/game/storage/contracts_campaign_participant_invite_character.go` -- CampaignReader/CampaignStore, ParticipantReader/ParticipantStore, CharacterReader/CharacterStore
- `internal/services/game/storage/contracts_session_scene.go` -- SessionReader/SessionStore, SessionGateReader/SessionGateStore, etc.
- `internal/services/game/storage/contracts_projection_state.go` -- CampaignReadStores, SessionReadStores, SceneReadStores, ProjectionStore

**Description:** Storage contracts consistently split into Reader (read-only) and Store (read + write) interfaces. Composite interfaces group related stores by concern. The doc comments consistently note "Projection handlers use the full interface; read-only consumers should prefer *Reader."

The composite interfaces (CampaignReadStores, SessionReadStores, SceneReadStores) compose 3-6 narrower interfaces each and are used by `ProjectionApplyTxStore` and `StoreBundle` for transaction-scoped work.

**Proposal:** No change. This is exemplary interface design that follows Go best practices.

---

### 16. DecisionGate is a value struct, not an interface

**Category:** not an issue (documented analysis)
**File:** `internal/services/game/domain/engine/gate.go:18-20`

**Description:** `DecisionGate` is a concrete struct with a `*command.Registry` field, not an interface. The Handler embeds it directly. `NewHandler` binds `Gate.Registry` from the shared `Commands` field to prevent drift.

This is fine: gate checking is pure logic with no I/O and no testability need for mocking. The struct provides a natural home for the `Check` and `CheckScene` methods.

**Proposal:** No change.

---

### 17. ReplayStateLoader is a concrete struct, not an interface

**Category:** not an issue (documented analysis)
**File:** `internal/services/game/domain/engine/loader.go:20-27`

**Description:** `ReplayStateLoader` is a concrete struct with 6 fields, implementing the `StateLoader` interface via its `Load` method. The Handler accepts `StateLoader` (interface), not `ReplayStateLoader` (concrete), preserving testability.

**Proposal:** No change. The Handler consumes the interface; the loader is one implementation.

---

### 18. CommandTyper interface position -- defined at consumption point

**Category:** not an issue (documented analysis)
**File:** `internal/services/game/domain/module/registry.go:81-83`

**Description:** `CommandTyper` is defined in the `module` package alongside `Module`, even though it is only consumed in `engine/registries_validation_system.go:120`. Per Go convention, interfaces should be at the consumption point.

However, `CommandTyper` is part of the game-system extension contract -- it tells system authors what to implement. Placing it in `module` (next to `Module`, `CharacterReadinessChecker`, `SessionStartBootstrapper`) keeps all extension-point interfaces co-located, which aids discoverability for system authors.

**Proposal:** Acceptable as-is. The discoverability benefit of co-location with the `Module` interface outweighs the Go convention of defining interfaces at consumption points in this case.

---

## Priority Summary

| # | Finding | Category | Priority |
|---|---------|----------|----------|
| 3a | CharacterReadinessChecker doc says optional but startup validation makes it mandatory | correctness risk | High |
| 4/14 | Applier StoreGroups half-migration -- flat struct + grouped construction | anti-pattern | Medium |
| 13 | Historical test seams without removal criteria | anti-pattern | Medium |
| 11 | Three registries with duplicate key/registration code | anti-pattern | Low |
| 3b | Optional interface discoverability for game-system authors | contributor friction | Low |
| 7 | Dual interface requirement (Module + GameSystem) undocumented | contributor friction | Low |
| 9 | StateFactory vs StateHandlerFactory naming | contributor friction | Low |
| 1 | Three Folder interfaces | contributor friction | Info |
| 10 | Speculative Healable/Damageable/ResourceHolder interfaces | anti-pattern | Info |
