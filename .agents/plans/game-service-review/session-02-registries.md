# Session 2: Command and Event Registries

## Status: `complete`

## Package Summaries

### `domain/command/registry.go` (357 lines)
Central registry for command definitions. Defines `Command` envelope struct, `Definition` metadata, `Registry` with `Register`, `ValidateForDecision`, `Definition`, and `ListDefinitions`. Handles validation, normalization, canonical JSON, system namespace enforcement, actor type defaults, and target entity extraction from payloads.

### `domain/event/registry.go` (521 lines)
Central registry for event definitions. Defines `Event` envelope struct, `Definition` metadata (with addressing policy and intent), `Registry` with `Register`, `ValidateForAppend`, `RegisterAlias`, `Resolve`, `ShouldFold`, `ShouldProject`, `MissingPayloadValidators`, and `ListDefinitions`. Richer than command registry due to: addressing policies, intent classification (projection vs replay-only vs audit), alias support for renamed events, and storage-field guards.

## Findings

### Finding 1: Registries Are Well-Structured — Not God Objects
- **Severity**: info
- **Location**: `domain/command/registry.go`, `domain/event/registry.go`
- **Issue**: The review plan questioned whether these are god-object registries. They are not. Each registry has a focused responsibility: store definitions, validate envelopes at a specific boundary (command before decision, event before append). The command registry is 357 lines and the event registry is 521 lines — both are within reasonable bounds for their responsibility.
- **Recommendation**: Current size and scope are appropriate. The registries are the right abstraction.

### Finding 2: No Thread Safety Documentation for Build Phase
- **Severity**: medium
- **Location**: `domain/command/registry.go:188`, `domain/event/registry.go:171`
- **Issue**: Both registries document "After initialization, all methods are safe for concurrent read access" but do not explicitly state that `Register` calls must happen during a single-goroutine build phase. The `Register` methods have no mutex — concurrent registration would race on the map. This is fine if the builder pattern enforces single-threaded setup, but a contributor unfamiliar with the bootstrap flow might call `Register` concurrently.
- **Recommendation**: Add a doc note: "Register must be called during single-goroutine initialization (before the registry is shared). It is not safe for concurrent use."

### Finding 3: Command and Event Registries Have Parallel But Not Shared Validation
- **Severity**: low
- **Location**: `domain/command/registry.go:228-295`, `domain/event/registry.go:226-269`
- **Issue**: Both registries implement similar validation logic (trim campaign ID, validate type, check owner, normalize system metadata, validate actor type, canonicalize JSON). The patterns are structurally identical but the details differ (commands validate target entity, events validate addressing policy and storage fields). Extracting shared logic would couple them unnecessarily.
- **Recommendation**: Keep separate. The parallel structure is a feature — each boundary has different invariants. Document that this is intentional in a contributing guide.

### Finding 4: Duplicate PayloadValidator Type Definition
- **Severity**: low
- **Location**: `domain/command/registry.go:179`, `domain/event/registry.go:119`
- **Issue**: Both packages define `type PayloadValidator func(json.RawMessage) error` independently. They are structurally identical but type-distinct.
- **Recommendation**: Could be unified into a shared type, but the duplication is minor and keeps packages independent. Leave as-is.

### Finding 5: Event Registry Alias System Is Well-Designed
- **Severity**: info
- **Location**: `domain/event/registry.go:432-477`
- **Issue**: The alias system (`RegisterAlias`, `Resolve`) handles event type renames cleanly. It validates that the canonical type exists before accepting an alias, prevents duplicate aliases, and `Resolve` transparently maps deprecated types. The documentation is excellent — it explains when to use aliases vs new types.
- **Recommendation**: This is a model for handling schema evolution in an event-sourced system.

### Finding 6: Event Registry Intent System Is Clean
- **Severity**: info
- **Location**: `domain/event/registry.go:136-150`
- **Issue**: The three-tier intent system (`ProjectionAndReplay`, `ReplayOnly`, `AuditOnly`) with `ShouldFold`/`ShouldProject` query methods cleanly separates event processing concerns. Default intent is `ProjectionAndReplay`, and the fail-closed behavior for unknown types is well-documented.
- **Recommendation**: Clean design. No changes needed.

### Finding 7: Registration API Is Clear for New Contributors
- **Severity**: info
- **Location**: `domain/command/registry.go:198-221`, `domain/event/registry.go:182-219`
- **Issue**: The `Register` method takes a `Definition` struct with clear fields. Error messages for misregistration are descriptive ("command type already registered: %s", "owner must be core or system"). The pattern is discoverable — find existing `Register` calls and follow the pattern.
- **Recommendation**: A contributing guide example showing "how to add a new command" would help, but the API itself is clear.

### Finding 8: Command Registry normalizeTargetEntity Silently Extracts from Payload
- **Severity**: low
- **Location**: `domain/command/registry.go:297-309`
- **Issue**: When `EntityID` is not set on the command and the definition specifies a `PayloadField`, the registry extracts it from the JSON payload silently. If the payload field doesn't exist or isn't a string, the entity ID remains empty with no error. This is intentional fallback behavior but could mask misconfiguration.
- **Recommendation**: Consider logging when payload extraction fails, or add a validation option to require the target entity for definitions that declare one.

### Finding 9: Event Registry MissingPayloadValidators Is a Useful Diagnostic
- **Severity**: info
- **Location**: `domain/event/registry.go:492-506`
- **Issue**: `MissingPayloadValidators` identifies event types that should have payload validation but don't. This is used by startup validation to catch registration gaps. Good defensive design.
- **Recommendation**: No changes needed. Consider a similar method for command registry.

### Finding 10: canonicalJSON Package-Level Var for Test Injection
- **Severity**: low
- **Location**: `domain/event/registry.go:49`
- **Issue**: `var canonicalJSON = coreencoding.CanonicalJSON` exists solely to let tests inject failures. This is a common Go pattern but means a production-visible package-level variable holds a function pointer that could theoretically be reassigned. The test `TestChainHash_ReturnsCanonicalJSONError` uses this.
- **Recommendation**: Acceptable pattern. An alternative would be to pass the function via a struct field, but the test-injection var is idiomatic in Go for leaf functions.

## Summary Statistics
- Files reviewed: 14 (2 production registry files + 5 production support files + 7 test files)
- Findings: 10 (0 critical, 0 high, 1 medium, 4 low, 5 info)
