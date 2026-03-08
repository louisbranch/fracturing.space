# GSR Phase 13: Core Utilities & Shared Packages

## Summary

Core packages demonstrate **clear design intent and strong isolation**. Dice/random are properly injectable for deterministic tests. No inverted dependencies. Platform could use better internal organization documentation. Shared packages are well-scoped, not a code dump.

## Findings

### F13.1: Core/Platform Boundary — Good

**Severity:** minor

Core = reproducible RPG mechanics (dice, checks, random seeding). Platform = infrastructure (I/O, transports, identity). Isolation is correct — core imports only `platform/errors`.

**Recommendation:** Add "Core Design Philosophy" guide explaining when to add to core/ vs domain/ vs domain/bridge/.

### F13.2: Shared Packages — Healthy Boundaries

**Severity:** style (no action needed)

15 packages across auth, transport, i18n, and web concerns. Average ~317 LOC each. No dumping-ground pattern. Each has narrow, intentional scope.

### F13.3: Dice/Random Injectability — Excellent

**Severity:** style (no action needed)

`RollDice()` deterministic with seed. `RollWithRng()` accepts injected `*rand.Rand`. `ResolveSeed()` accepts `seedFunc` and `allowClientSeed` callbacks. No global state.

### F13.4: ID Generation — Sound

**Severity:** minor

`id.NewID()` uses `crypto/rand` for UUIDv4 base32 encoding. Metadata interceptor accepts injected `idGenerator func() (string, error)` for test overrides.

**Recommendation:** Audit all services for consistent ID injection pattern.

### F13.5: No Inverted Dependencies — Clean

**Severity:** style (no action needed)

Only import from shared → game service: `grpcauthctx` → `api/grpc/metadata` (transport constants, not domain logic). Acceptable.

### F13.6: Utility Growth — Managed

**Severity:** minor

Platform (7,711 LOC across 14 directories) lacks top-level organization guide.

**Recommendation:** Add `platform/doc.go` categorizing sub-packages by purpose (Transport, Domain Infrastructure, Presentation, Observability).

## Cross-References

- **Phase 1** (Package Structure): Core package assessment
- **Phase 11** (Configuration): Platform timeouts, serviceaddr usage
- **Phase 12** (Testing): Test determinism via injectable packages
