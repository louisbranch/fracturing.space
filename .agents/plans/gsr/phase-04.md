# GSR Phase 4: Command/Decision Pattern

## Summary

The command/decision architecture is **sound and well-maintained**. Deciders are pure functions with no I/O, rejection codes follow consistent conventions, and gate enforcement is centralized. Minor improvements around DecideFunc variant documentation and route table consolidation would reduce cognitive load.

## Findings

### F4.1: DecideFunc Variant Zoo — Medium

**Severity:** important

Four variants exist: `DecideFunc`, `DecideFuncWithState`, `DecideFuncTransform`, `DecideFuncMulti`. Adoption is partial — many domain deciders still use raw switch statements. `DecideFuncTransform` exists but is unused.

**Recommendation:** Document adoption criteria (e.g., "use DecideFunc for single-event simple payload; use raw switch for multi-event or complex validation"). Remove `DecideFuncTransform` if no use case emerges.

### F4.2: Route Table Dual Maintenance — Minor

**Severity:** minor

`staticCoreCommandRoutes()` and `buildCoreRouteTable()` both define routing. Startup validation catches mismatches, but it's still dual maintenance.

**Recommendation:** Consider generating core routes from registered command definitions rather than maintaining the static table separately.

### F4.3: Type Assertion Ceremony — Sound

**Severity:** style (no action needed)

All deciders receive `any` state and extract typed state at the routing boundary. Pattern is defensive and consistent across domains.

### F4.4: Rejection Code Conventions — Excellent

**Severity:** style (no action needed)

Consistent domain-prefixed SCREAMING_SNAKE codes (e.g., `SESSION_NOT_ACTIVE`, `PARTICIPANT_NOT_FOUND`). Well-documented and stable.

### F4.5: Decider Purity — Excellent

**Severity:** style (no action needed)

All deciders are pure functions — no I/O, no time access, no context dependency. Fully replay-friendly.

### F4.6: sessionStartRoute Cross-Domain Events — Acceptable

**Severity:** minor

`sessionStartRoute` emits events across aggregate boundaries. This is a documented exception for atomicity. Acceptable but worth noting for future contributors.

### F4.7: Entity ID Resolution — Documentation Gap

**Severity:** minor

Pattern for extracting entity IDs from command payloads (e.g., `participantStateFor`) is solid with defensive null-checks and zero-value fallbacks, but lacks top-level documentation explaining the convention.

**Recommendation:** Add a doc comment or guide explaining when to use EntityID vs. payload fields in command routing.

### F4.8: GatePolicy Coverage — Complete, Could Be Explicit

**Severity:** style

Gate enforcement is centralized in `DecisionGate`. Session and scene gates are fully covered. Action commands (rolls, notes) implicitly pass through (absence = no gate), which is correct behavior but could be more explicit with `GateScopeNone` declarations.

**Recommendation:** Register action commands with `GateScopeNone` to clarify intent.

## Strengths

- Pure domain logic: deciders are deterministic and replay-friendly
- Rejection convention is stable, domain-prefixed, with clear semantics
- `NewEvent()` copies envelope consistently; no ad-hoc event construction
- Gate enforcement centralized in `DecisionGate`, not scattered across domains
- Startup validation catches route table mismatches early

## Cross-References

- **Phase 3** (Event System): Event emission patterns from deciders
- **Phase 5** (Engine Orchestration): Gate evaluation ordering
- **Phase 9** (Module Extension): Bridge deciders follow same patterns
