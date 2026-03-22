# Pass 11: Testing Infrastructure and Coverage

## Summary

The testing infrastructure is surprisingly well-structured for a codebase of this
size. The project uses a layered test strategy (unit, conformance, integration,
Lua scenario) with architecture contract tests enforcing import boundaries and
doc.go conventions. Coverage floors are per-package with a ratchet mechanism, and
a module conformance testkit validates fold/decider/adapter coverage
automatically for each system module.

The primary issues are: (1) heavy duplication of in-memory store fakes across
three separate packages, (2) several transport/handler packages with zero direct
tests relying entirely on integration/scenario coverage, (3) the join-grant JWT
builder is duplicated between two files with identical logic, (4) the
`projection/testevent` package defines an independent Event type that mirrors the
real `domain/event.Event` creating sync-drift risk, and (5) there are no
property-based or fuzz tests anywhere in the codebase.

---

## Findings

### 1. Triple-duplicated in-memory DaggerheartStore fakes

**Category:** anti-pattern
**Severity:** medium
**Files:**
- `internal/services/game/api/grpc/game/gametest/fakes_daggerheart.go` (`FakeDaggerheartStore`, ~338 lines)
- `internal/test/mock/gamefakes/stores.go` (`DaggerheartStore`, duplicated DH methods within ~700 line file)
- `internal/services/game/domain/module/testkit/conformance_test.go` (`memDaggerheartStore`, ~230 lines)

Three independent in-memory implementations of `projectionstore.Store` exist with
near-identical logic. Each implements the same interface methods with minor
differences in key construction (`"/"` vs `":"` separator, concurrency safety
via `sync.Mutex` in only one). A fourth variant
(`fakeDaggerheartStore`) exists in
`internal/services/game/api/grpc/systems/daggerheart/actions_store_helpers_test.go`
as a type alias for `gamefakes.DaggerheartStore`.

**Key differences:**
- `gametest.FakeDaggerheartStore`: uses `map[string]map[string]` (campaignID -> entityID), tracks `StatePuts`/`SnapPuts` counters
- `gamefakes.DaggerheartStore`: uses flat `map[string]` with `":"` composite keys, no counters
- `memDaggerheartStore` (conformance_test.go): uses `map[string]` with `"/"` composite keys, has `sync.Mutex`, has `Reset()` method

**Proposal:** Consolidate into a single canonical `projectionstore.MemStore` in a
shared test support package (possibly `projectionstore` itself with a `_test.go`
export or a dedicated `projectionstore/memstore` package). The variant that the
conformance testkit needs (thread-safe + Reset) could be the canonical one, with
others delegating to it.

---

### 2. Duplicated core store fakes (Campaign, Character, Session, Event)

**Category:** anti-pattern
**Severity:** medium
**Files:**
- `internal/services/game/api/grpc/game/gametest/fakes_campaign.go` (and siblings `fakes_character.go`, `fakes_session.go`, `fakes_event.go`)
- `internal/test/mock/gamefakes/stores.go` (parallel `CampaignStore`, `CharacterStore`, `SessionStore`, `EventStore`)

The same duplication pattern as finding #1, but for the core storage interfaces.
`gametest` fakes use nested maps (`map[string]map[string]`) while `gamefakes`
uses flat maps with composite keys. Both implement the same storage interfaces
with semantically identical behavior.

Additionally, `gametest` provides both `FakeSessionGateStore`,
`FakeSessionSpotlightStore`, and `FakeSessionInteractionStore` while
`actions_store_helpers_test.go` re-implements
`fakeSessionGateStore`/`fakeSessionSpotlightStore` locally with slightly
different behavior (e.g., the local one has no state tracking).

**Proposal:** Same as #1 -- consolidate. The `gametest` variants are more
featureful (counters, per-method error injection) and should likely be the
canonical ones.

---

### 3. Duplicated join-grant JWT builder

**Category:** anti-pattern
**Severity:** low
**Files:**
- `internal/services/game/api/grpc/game/gametest/helpers.go:113` (`JoinGrantSigner.Token`)
- `internal/test/integration/harness_test.go:379` (`joinGrantToken`)

Both functions construct identical EdDSA JWT tokens with the same claim
structure. The integration harness version uses package-level `joinGrantPrivateKey`
while the `gametest` version uses a struct-held key, but the encoding logic is
copy-pasted.

**Proposal:** Have the integration harness construct a `gametest.JoinGrantSigner`
and call its `Token()` method instead of duplicating the JWT construction.

---

### 4. `projection/testevent` defines a parallel Event type with sync-drift risk

**Category:** correctness risk
**Severity:** medium
**File:** `internal/services/game/projection/testevent/event.go`

This package defines its own `Event` struct and `Type` constants that mirror
`domain/event.Event` and the domain event type constants. The projection applier
tests use `testevent.Event` rather than the canonical `domain/event.Event`. If
fields are added to or removed from the canonical event, this parallel type will
silently drift.

The `Type` constants (e.g., `TypeCampaignCreated`, `TypeParticipantJoined`) are
string values that must match the canonical event types but have no compile-time
guarantee of parity.

**Proposal:** Either:
(a) Remove the parallel Event struct and have applier tests use `domain/event.Event` directly, or
(b) Add a contract test that asserts field-by-field parity between `testevent.Event` and `domain/event.Event`, plus constant parity for all Type values.

---

### 5. No property-based or fuzz tests

**Category:** missing best practice
**Severity:** medium
**Scope:** entire codebase

There are zero `testing/quick`, `pgregory.net/rapid`, or `testing.F` (fuzz)
tests in the game service. For an event-sourced system, property-based tests
would be valuable for:

- **Fold/replay determinism:** Given any sequence of valid events, folding them
  produces the same state as replaying through the snapshot mechanism.
- **Command idempotency properties:** Certain commands should be idempotent when
  applied twice.
- **Serialization round-trip:** Event `PayloadJSON` serialization and
  deserialization round-trips correctly for all event types.
- **Hash chain integrity:** Generating events, persisting, and reading back
  preserves the cryptographic hash chain.

The conformance testkit's `validateFoldIdempotency` (line 144 of
`conformance.go`) is the closest existing mechanism, but it tests only a single
event applied to fresh state, not arbitrary sequences.

**Proposal:** Add at minimum a fold/replay determinism property test for the
Daggerheart system: generate random valid event sequences, fold them into state,
snapshot, then replay and verify equivalence.

---

### 6. Several transport/handler packages have zero test files

**Category:** contributor friction
**Severity:** low-medium
**Packages without `_test.go` files:**
- `api/grpc/game/handler/` (610 lines: actor resolution, auth, domain helpers, pagination, social)
- `api/grpc/game/characterworkflow/` (transport wiring)
- `api/grpc/systems/daggerheart/environmenttransport/`
- `api/grpc/systems/daggerheart/gmconsequence/`
- `api/grpc/systems/daggerheart/guard/`
- `api/grpc/systems/daggerheart/statmodifiertransport/`

These packages contain production logic (the `handler` package is 610 lines of
shared helpers) that is presumably covered by tests in consuming packages and
integration/scenario tests, but has no direct unit tests. For the `handler`
package specifically, functions like `ResolveActorFromMetadata`,
`RequireCampaignAccess`, `PaginateSlice`, and `MatchAvatarToProfile` contain
branching logic that would benefit from direct unit tests.

**Proposal:** Add targeted unit tests for `handler/` package functions that
contain branching logic. The single-file transport wiring packages
(`characterworkflow`, `guard`, `gmconsequence`) are acceptable without direct
tests if covered by integration tests.

---

### 7. Critical domain internals tested only through parent package

**Category:** contributor friction
**Severity:** low
**Packages:**
- `domain/systems/daggerheart/internal/decider/` (2,274 lines, 17 source files, 0 test files)
- `domain/systems/daggerheart/internal/folder/` (798 lines, 11 source files, 0 test files)
- `domain/systems/daggerheart/internal/validator/` (1,498 lines, 12 source files, 0 test files)

These packages contain the core game mechanics logic (command decision handlers,
event fold handlers, payload validators) and have zero `_test.go` files in their
own directories. They are tested through the parent `daggerheart` package's tests
(confirmed by imports in `daggerheart/*_test.go` files) and indirectly through
224 Lua scenario tests.

The coverage floors file confirms these packages have coverage:
- `internal/decider`: floor 86.0%
- `internal/validator`: floor 90.0%
- `internal/folder`: covered via `daggerheart` floor 91.5%

This is acceptable per the testing policy ("use the right level of tests"), but
the lack of co-located tests means contributors must navigate to the parent
package to find relevant tests, which is a discoverability friction point.

**Proposal:** No immediate action needed given the coverage floors are
maintained. Consider adding a `_test.go` in each package with a comment pointing
to where the canonical tests live.

---

### 8. Storage contracts package (971 lines) has no tests

**Category:** missing best practice
**Severity:** low
**File:** `internal/services/game/storage/` (8 source files, 0 test files)

The storage package defines 8+ store interfaces (~971 lines including record
types). While interfaces themselves don't need unit tests, the record types and
sentinel errors defined here have no compile-time contract verification. A
contract test could assert that all `Err*` sentinels conform to `errors.Is`
expectations and that record types are JSON-serializable.

**Proposal:** Add a minimal `storage_test.go` with sentinel-error contract
assertions.

---

### 9. `testevent` package is only consumed by projection applier tests

**Category:** contributor friction
**Severity:** low
**Files:**
- `internal/services/game/projection/testevent/event.go` (consumed by 7 test files)

The `testevent` package exists solely for the projection applier test suite. Its
doc.go says "test-only projection event fixtures", which is accurate. However,
the package sits under `projection/` at the same level as production code, which
may confuse contributors into thinking it is production infrastructure.

**Proposal:** No structural change needed since the `doc.go` is clear and the
package naming includes "test". The sync-drift risk (finding #4) is the
more actionable concern.

---

### 10. Module testkit conformance is comprehensive but single-system

**Category:** missing best practice
**Severity:** low
**File:** `internal/services/game/domain/module/testkit/conformance_test.go`

`ValidateSystemConformance` is a strong second-system-author tool: it checks fold
coverage, decider command coverage, adapter event coverage, router definition
parity, state factory determinism, fold idempotency, and adapter snapshot smoke.
`ValidateAdapterIdempotency` adds duplicate-event resilience checks.

However, the conformance test only exercises the Daggerheart system. When a
second game system is added, the test must be manually duplicated with a new
`TestValidateSystemConformance_<System>` function and a new in-memory store
implementation. The testkit itself is well-designed for this (it accepts generic
`module.Module` and `bridge.Adapter` interfaces), but the test file contains no
table-driven structure that would automatically pick up new systems from the
manifest.

**Proposal:** When a second system is added, consider a table-driven test that
iterates over `systemmanifest.Modules()` and validates each system's conformance.
This requires each system to provide a test-store constructor, which could be
formalized as part of the module interface or a parallel test-registry.

---

### 11. Architecture contract tests are effective but narrowly scoped

**Category:** missing best practice
**Severity:** low
**File:** `internal/services/game/domain/internaltest/contracts/architecture_contracts_test.go`

Three contract tests exist:
1. `TestGamePackagesHaveDocGo` -- enforces doc.go presence
2. `TestGameDocGoUsesPackageCommentConvention` -- enforces "Package X ..." format
3. `TestDomainImportsRespectArchitectureContracts` -- enforces domain layer isolation

These are well-implemented using AST parsing. The import contract test has an
explicit allowlist for packages permitted to import storage/proto packages.
However, there are no contract tests for:
- Transport layer import isolation (handler packages should not import storage directly)
- Circular dependency detection beyond what the Go compiler provides
- Package naming conventions (e.g., `*transport` suffix for transport packages)

**Proposal:** Consider adding a transport-layer import contract that prevents
`api/grpc/*` packages from importing `storage/sqlite/*` directly.

---

### 12. Write-path architecture guard is well-factored

**Category:** (positive observation)
**File:** `internal/services/game/domain/module/testkit/arch_guard.go`

The `ValidateWritePathArchitecture` function uses AST scanning to enforce three
invariants across handler files: no inline `.Apply` calls, no direct storage
mutations, and no forbidden string literals. This is configurable via
`WritePathPolicy` struct and has its own unit tests
(`arch_guard_test.go`). This is a high-quality architecture enforcement pattern.

---

### 13. Integration test harness contains duplicated env-setup helpers

**Category:** anti-pattern
**Severity:** low
**File:** `internal/test/integration/harness_test.go`

The integration harness has two parallel code paths for environment setup:
- `setJoinGrantEnv` (uses `t.Setenv`, scoped to test)
- `setJoinGrantProcessEnv` (uses `os.Setenv`, process-wide)
- `setAISessionGrantEnv` / `setAISessionGrantProcessEnv` (same pattern)
- `setTempDBPath` / `setTempAuthDBPath` (delegate to testkit)

The `*ProcessEnv` variants exist to support shared fixtures across test
functions. While functional, the two parallel paths risk diverging and make it
unclear which env vars are required.

**Proposal:** Consider a single env-setup function that takes a setter function
(`func(string, string) error`), similar to what `testkit.SetGameDBPaths` already
does, to eliminate the parallel implementations.

---

### 14. Coverage floor mechanism is well-designed

**Category:** (positive observation)
**File:** `docs/reference/coverage-floors.json`

The per-package coverage floors with a 0.1% allowed drop tolerance and a ratchet
tool (`make coverage-floors-ratchet`) is a mature approach to coverage
management. 40+ packages are tracked with floors ranging from 57% (AI SQLite
storage) to 100% (web module guardrails). Critical domain packages have floors
above 86%.

---

### 15. Lua scenario test corpus is extensive but has no manifest completeness check

**Category:** missing best practice
**Severity:** low
**Files:**
- `internal/test/game/scenarios/` (224 Lua files)
- `internal/test/game/scenarios/manifests/smoke.txt` (52 entries)

224 scenario files exist but the smoke manifest only includes 52. There is no
contract test verifying that all scenario files are reachable from at least one
manifest, which means scenarios can be added and never run in CI.

**Proposal:** Add a contract test that lists all `*.lua` files under
`systems/` and asserts each appears in at least one manifest file, or is
explicitly marked as draft/disabled.

---

### 16. All test doubles are hand-written (no code generation)

**Category:** (observation, not a deficiency)
**Scope:** entire codebase

The project uses zero code generation for test doubles -- no `mockgen`,
`counterfeiter`, `moq`, or similar tools. All fakes are hand-written
implementations of store interfaces. The single `go:generate` directive in the
game domain is for event documentation generation, not mocks.

This is a deliberate design choice that avoids tool dependencies and gives full
control over fake behavior (e.g., error injection, call counting). The tradeoff
is the maintenance burden evidenced by finding #1/#2 (triple-duplicated store
fakes).

---

### 17. `EqualSlices` in contracts package uses reflect.DeepEqual unnecessarily

**Category:** anti-pattern
**Severity:** very low
**File:** `internal/services/game/domain/internaltest/contracts/slices.go:11`

```go
func EqualSlices[T comparable](left, right []T) bool {
    return reflect.DeepEqual(left, right)
}
```

Since `T` is constrained to `comparable`, a simple loop comparison would avoid
the `reflect` import and be more idiomatic. `reflect.DeepEqual` is
appropriate for non-comparable types but is overkill here.

**Proposal:** Replace with a loop-based comparison:
```go
func EqualSlices[T comparable](left, right []T) bool {
    if len(left) != len(right) { return false }
    for i := range left {
        if left[i] != right[i] { return false }
    }
    return true
}
```

---

### 18. `SequentialIDGenerator` has a character encoding bug

**Category:** correctness risk
**Severity:** low
**File:** `internal/services/game/api/grpc/game/gametest/helpers.go:51`

```go
func SequentialIDGenerator(prefix string) func() (string, error) {
    counter := 0
    return func() (string, error) {
        counter++
        return prefix + "-" + string(rune('0'+counter)), nil
    }
}
```

For `counter > 9`, `rune('0'+counter)` produces non-digit Unicode characters
(`:`, `;`, `<`, etc.). At `counter = 10`, the ID becomes `prefix-:` instead of
`prefix-10`. This works for tests that generate fewer than 10 IDs, but will
produce confusing results otherwise.

**Proposal:** Use `fmt.Sprintf("%s-%d", prefix, counter)` or
`strconv.Itoa(counter)`.

---

### 19. `min` function in testkit shadows Go 1.21+ builtin

**Category:** anti-pattern
**Severity:** very low
**File:** `internal/test/testkit/runtime.go:141`

```go
func min(a, b time.Duration) time.Duration {
    if a < b { return a }
    return b
}
```

Go 1.21+ includes a builtin `min` function. This local definition shadows it.
If the project's minimum Go version is 1.21+, this can be deleted.

**Proposal:** Remove the function if go.mod requires Go >= 1.21.

---

### 20. Scenario builder defaults to Daggerheart, limiting reuse for second systems

**Category:** contributor friction
**Severity:** low
**File:** `internal/services/game/api/grpc/game/gametest/scenario_builder.go:72`

`NewCampaignScenario` defaults `system` to `systems.SystemIDDaggerheart`. When a
second game system is added, every test using the builder for that system must
explicitly call `.WithSystem()`. The builder also hard-constructs a
`FakeDaggerheartStore` in `Build()` (line 166), which will need to be
generalized.

**Proposal:** When a second system is implemented, extract the
system-specific store construction into the builder configuration so
`CampaignScenario.Daggerheart` becomes one of N optional system stores.
