# Anti-Patterns and Code Smells

Living document of anti-patterns found during the codebase audit (2026-02-13).
Items marked **[Fixed]** were resolved during the refactoring; the rest remain
as guidance to prevent recurrence.

## Resolved

### 1. Copy-paste utility functions [Fixed]

`EnvLookup` type and `envOrDefault` duplicated 6 times across cmd packages.

**Fix**: Migrated all config loading to `caarlos0/env` struct tags. Deleted
`internal/platform/config.EnvOrDefault` and `EnvLookup` — no longer needed.

### 2. Runtime nil checks for compile-time invariants [Fixed]

107 store nil checks in gRPC handlers that should be validated at construction
time. Every handler method began with `if s.stores.X == nil { return error }`.

**Fix**: Added `Stores.Validate()` called at service construction. Removed all
107 individual handler nil checks and 33 corresponding MissingStore tests.

### 3. God method [Fixed]

`Applier.Apply()` was 1470 lines with 24 switch cases. Each case followed the
same pattern (unmarshal payload, parse enums, persist to stores) but was not
decomposed.

**Fix**: Split into 7 domain-grouped files (apply_campaign.go,
apply_participant.go, etc.). The dispatcher is now ~85 lines.

### 4. Duplicate enum parsing [Fixed]

10 `parseXxx` functions in applier.go duplicated logic that domain types should
own. The applier and gRPC helpers each maintained their own copies.

**Fix**: Added canonical `FromLabel()` functions to domain packages (campaign,
participant, character). Applier delegates to these.

### 5. God object (game Server) [Fixed]

Game `Server` struct held 7 unrelated fields (listener, gRPC server, health,
3 SQLite stores, auth connection).

**Fix**: Extracted `storageBundle` type that groups the 3 stores with
`Open()`/`Close()` lifecycle. Server reduced to 5 fields.

### 6. Ad-hoc Applier construction [Fixed]

25+ places constructed `projection.Applier{}` with manually-selected store
fields. Easy to forget a store or wire the wrong one.

**Fix**: Added `Stores.Applier()` factory methods on both `gamegrpc.Stores`
and `daggerheart.Stores`. All handler constructions replaced.

### 7. Inconsistent signal handling [Fixed]

Servers used `signal.NotifyContext()`, CLI tools used a manual channel pattern.

**Fix**: Standardized all CLI tools (seed, scenario, maintenance) on
`signal.NotifyContext`. Servers already used this pattern.

### 8. Scattered defaults [Fixed]

Config defaults lived in both `internal/cmd/` and `internal/services/*/app/`.
No single source of truth per service.

**Fix**: Migrated all config loading to `caarlos0/env` struct tags with
`envDefault:` annotations. Defaults are now declarative and co-located with
the struct fields they apply to.

### 9. Environment variable naming inconsistency [Fixed]

Mix of `_ADDR`, `_HTTP_ADDR`, `_DB_PATH` suffixes with no documented
convention.

**Fix**: Established `FRACTURING_SPACE_{SERVICE}_{COMPONENT}_{SUFFIX}` naming
convention. All env vars now follow this pattern via struct tags.

### 10. Hardcoded timeouts [Fixed]

`2 * time.Second` gRPC dial timeout appeared in 3+ places. `5 * time.Second`
read header and shutdown timeouts duplicated across admin/web servers.
MCP domain had 32 inline timeout literals.

**Fix**: Created `internal/platform/timeouts` package with shared constants
(`GRPCDial`, `GRPCRequest`, `ReadHeader`, `Shutdown`). MCP domain uses
package-level `grpcCallTimeout` and `grpcLongCallTimeout`.

### 11. Manual JWT parsing [Fixed]

~200 lines of hand-rolled JWT validation in `invite/join_grant.go`.

**Fix**: Migrated to `golang-jwt/jwt/v5` which handles EdDSA natively.

### 12. Inconsistent error exit patterns [Fixed]

Servers used `log.Fatalf()`, tools used `fmt.Fprintf(os.Stderr) + os.Exit(1)`.

**Fix**: Created `config.Exitf` helper. All CLI entry points now use it.

### 13. No context deadlines in CLI tools [Fixed]

seed, scenario, and maintenance accepted context but never set deadlines.

**Fix**: Added configurable timeout (default 10m) to seed and maintenance via
`env:` tag and `-timeout` flag. Context deadline wraps the signal context.

## Open

### 14. Generic field maps

`CampaignUpdatedPayload.Fields` uses `map[string]any` requiring string-based
field checks at every consumption site. Consider typed update structs per field
group to catch mismatches at compile time.

### 15. Anemic domain types

Most domain types (Campaign, Participant, Character, Session) are pure data
with logic in package-level functions. Acceptable for now, but if behavior
grows, attach methods to the types.

### 16. Fat transport handlers

gRPC handlers orchestrate validation, event construction, persistence, and
external service calls in a single method (for example
`internal/services/game/api/grpc/game/campaign_service.go`). Extract an
application-layer command handler to keep transport thin and testable.

### 17. Stringly-typed event payload mapping in transport

Transport code builds event payloads directly from enum `.String()` values,
which hides validation and makes payload evolution harder. Prefer domain or
application-layer factories that validate and centralize payload mapping.

### 18. No-op event emissions

`character.profile_updated` can be emitted even when the payload matches the
current projection state. This makes timelines noisy and complicates change
summaries. Consider guarding against no-op updates in the game service layer.

### 19. Silent env parsing failures

Multiple server bootstraps ignore `env.Parse` errors (for example auth/game/admin).
This can mask misconfiguration and make failures non-obvious. Surface parse
errors early and fail fast.

### 20. Monolithic server constructors

Service constructors mix config loading, IO, dependency wiring, and runtime
startup logic in a single function. This makes unit testing difficult and
encourages global side effects. Split config, wiring, and runtime concerns.

### 21. Duplicated gRPC dial and health check logic

Admin, game, and MCP each implement custom gRPC dial, health checks, and retry
loops. This duplication increases drift and inconsistent behavior. Centralize
in a shared platform helper with test seams.

### 22. Oversized transport mappers

`internal/services/game/api/grpc/game/helpers.go` is a large conversion hub that
mixes multiple domains, making changes risky and hard to test. Split by domain
and keep conversions close to their types.
