# Pass 2: Transport Layer Structural Duplication

## Summary

The transport layer follows a well-factored layered pattern: gRPC service
structs delegate to `*Application` types that coordinate reads, authorization,
command building, and domain writes. A shared infrastructure stack
(`commandbuild`, `domainwrite`, `handler`, `grpcerror`, `validate`, `authz`)
absorbs most cross-cutting concerns. However, structural duplication has
accumulated in three distinct tiers:

1. **System-transport contract explosion** -- every Daggerheart transport
   subpackage redeclares identical narrow interfaces (`CampaignStore`,
   `SessionGateStore`, `DaggerheartStore`) and an identical
   `DomainCommandInput`/`SystemCommandInput` struct. There are 11 copies of
   `CampaignStore interface`, 12 copies of `SessionGateStore interface`, and
   10 copies of `DomainCommandInput`/`SystemCommandInput` struct across
   daggerheart subpackages.

2. **Handler write-path ceremony** -- the canonical write sequence (resolve
   actor, marshal payload, build command, execute-and-apply, reload record) is
   repeated with minor variations across 27+ call sites. Session transport
   partially addressed this with a local `sessionCommandExecutor`, but other
   entity transports repeat the full inline sequence.

3. **Error normalization inconsistency** -- core-game transports route errors
   through `handler.ExecuteAndApplyDomainCommand` (which calls
   `grpcerror.EnsureStatus`), while system transports use
   `grpcerror.HandleDomainError` for store/domain errors and inline
   `status.Error(...)` for precondition failures. The two paths produce
   different gRPC status codes for equivalent domain errors.

---

## Findings

### F01 -- System-transport `DomainCommandInput`/`SystemCommandInput` struct duplication
**Category:** anti-pattern
**Severity:** medium

The same 8-10 field struct is declared independently in 10 system transport
packages:

| File | Struct |
|------|--------|
| `systems/daggerheart/damagetransport/contracts.go:47` | `SystemCommandInput` |
| `systems/daggerheart/conditiontransport/contracts.go:37` | `DomainCommandInput` |
| `systems/daggerheart/adversarytransport/contracts.go:41` | `DomainCommandInput` |
| `systems/daggerheart/gmmovetransport/contracts.go:50` | `DomainCommandInput` |
| `systems/daggerheart/statmodifiertransport/contracts.go:30` | `DomainCommandInput` |
| `systems/daggerheart/recoverytransport/contracts.go:33` | `SystemCommandInput` |
| `systems/daggerheart/countdowntransport/contracts.go:34` | `DomainCommandInput` |
| `systems/daggerheart/outcometransport/contracts.go:60` | `SystemCommandInput` |
| `systems/daggerheart/environmenttransport/contracts.go:34` | `DomainCommandInput` |
| `systems/daggerheart/workflowruntime/contracts.go:35` | `SystemCommandInput` |

The naming is also inconsistent: some packages call it `SystemCommandInput`,
others `DomainCommandInput`. All carry the same fields: `CampaignID`,
`CommandType`, `SessionID`, `SceneID`, `RequestID`, `InvocationID`,
`EntityType`, `EntityID`, `PayloadJSON`, `MissingEventMsg`, `ApplyErrMessage`.

**Proposal:** Extract a single `SystemCommandInput` struct to a shared
package (e.g. `api/grpc/systems/daggerheart/transportcontracts/` or co-locate
in the existing `guard` package). Each transport package would import instead
of redeclaring.

---

### F02 -- `CampaignStore` interface redeclared 11 times in system transports
**Category:** anti-pattern
**Severity:** medium

The exact same single-method interface:
```go
type CampaignStore interface {
    Get(ctx context.Context, id string) (storage.CampaignRecord, error)
}
```
is declared in 11 different `contracts.go` files across daggerheart
subpackages. Similarly, `SessionGateStore` (single method
`GetOpenSessionGate`) appears 12 times, and `DaggerheartStore` (varying
method sets) appears 11 times.

**Files (CampaignStore):**
- `conditiontransport/contracts.go:13`
- `adversarytransport/contracts.go:14`
- `environmenttransport/contracts.go:13`
- `charactermutationtransport/contracts.go:14`
- `recoverytransport/contracts.go:13`
- `statmodifiertransport/contracts.go:12`
- `countdowntransport/contracts.go:12`
- `gmmovetransport/contracts.go:14`
- `damagetransport/contracts.go:14`
- `sessionrolltransport/contracts.go:11`
- `outcometransport/contracts.go:15`

**Proposal:** The `guard` package already centralizes `SessionGateStore`.
Promote `CampaignStore` and a superset `DaggerheartReadStore` interface to
the same shared contracts location. Individual packages can embed or alias as
needed. This follows the project convention of defining interfaces at
consumption points, but "11 identical consumption points" suggests the
interface belongs one level up.

---

### F03 -- `requireDependencies` nil-check boilerplate in every system handler
**Category:** contributor friction
**Severity:** low

18 distinct `require*Dependencies()` methods manually nil-check each
`Dependencies` struct field with a switch-case per field:

- `conditiontransport/handler.go:429` `requireCharacterDependencies`
- `conditiontransport/handler.go:446` `requireAdversaryDependencies`
- `adversarytransport/handler.go:92` `requireBaseDependencies`
- `charactermutationtransport/handler_support.go:16` `requireDependencies`
- `gmmovetransport/handler.go:184` `requireDependencies`
- `statmodifiertransport/handler.go:184` `requireDependencies`
- `damagetransport/handler.go:269` `requireDamageDependencies`
- `damagetransport/handler_adversary.go:184` `requireAdversaryDamageDependencies`
- `sessionrolltransport/handler.go:19-88` (4 variants)
- `recoverytransport/handler_helpers.go:8` `requireDependencies`
- `environmenttransport/handler.go:352` `requireBaseDependencies`
- `outcometransport/handler_helpers.go:321-338` (2 variants)
- `countdowntransport/handler_support.go:16` `requireDependencies`

Each method checks 3-7 fields with identical `status.Error(codes.Internal, "... is not configured")` patterns. The checks protect against nil-pointer
panics in production, but the boilerplate is significant.

**Proposal:** Use a generic `RequireDeps` helper that accepts `any` tagged
struct fields and produces standardized error messages. Alternatively,
enforce non-nil at constructor time (`NewHandler`) with `panic` or
constructor-level error return, eliminating the need for per-call checks
entirely. The constructor approach is idiomatic Go for mandatory
dependencies.

---

### F04 -- Inline write-path ceremony in core-game transport methods
**Category:** contributor friction
**Severity:** medium

The canonical write sequence appears inline in 27+ call sites across core-game
transport:

```go
actorID, actorType := handler.ResolveCommandActor(ctx)
payloadJSON, err := json.Marshal(payload)
if err != nil { return ..., grpcerror.Internal("encode payload", err) }
_, err = handler.ExecuteAndApplyDomainCommand(
    ctx, c.write, c.applier,
    commandbuild.Core(commandbuild.CoreInput{
        CampaignID:   campaignID,
        Type:         handler.CommandType...,
        ActorType:    actorType,
        ActorID:      actorID,
        RequestID:    grpcmeta.RequestIDFromContext(ctx),
        InvocationID: grpcmeta.InvocationIDFromContext(ctx),
        EntityType:   "entity",
        EntityID:     entityID,
        PayloadJSON:  payloadJSON,
    }),
    domainwrite.Options{...},
)
```

The sessiontransport package introduced `sessionCommandExecutor` (see
`session_command_execution.go:41`) to collapse this to a single method call,
proving the pattern is extractable. Other entity transports still inline it.

**Representative sites:**
- `campaigntransport/campaign_mutation_application.go:65-81`
- `campaigntransport/campaign_status_application.go:35-50, 78-93, 117-132`
- `campaigntransport/campaign_ai_binding_application.go:47-63`
- `scenetransport/scene_lifecycle_application.go:72-90, 132-149, 181-198, 254-271`
- `participanttransport/participant_create_application.go:134-152`
- `charactertransport/character_create_application.go:125-141`

**Proposal:** Generalize `sessionCommandExecutor` into a shared
`EntityCommandExecutor` in the `handler` package. Each entity application
would construct one during initialization, reducing every write call site to
a single `executor.Execute(ctx, input)`. The `input` struct would carry
`CommandType`, `CampaignID`, `EntityType`, `EntityID`, `SessionID` (optional),
`SceneID` (optional), `Payload` (any), and `Options`.

---

### F05 -- `timestampOrNil` duplicated across transport mappers
**Category:** anti-pattern
**Severity:** low

The identical function:
```go
func timestampOrNil(value *time.Time) *timestamppb.Timestamp {
    if value == nil { return nil }
    return timestamppb.New(*value)
}
```
is independently defined in:
- `campaigntransport/mappers.go:171`
- `sessiontransport/mappers.go:139`

while `handler/mappers.go:14` already exports `TimestampOrNil` with the same
logic (plus a `.UTC()` call).

**Proposal:** Delete the private copies and use `handler.TimestampOrNil`
everywhere. Note the `.UTC()` normalization difference -- ensure both callers
are OK with UTC timestamps (they should be, since proto timestamps are
UTC-based).

---

### F06 -- Error normalization divergence between core-game and system transports
**Category:** correctness risk
**Severity:** high

Core-game transports route all write errors through
`handler.ExecuteAndApplyDomainCommand`, which calls
`domainwrite.TransportExecuteAndApply` (with `NormalizeDomainWriteOptions`)
and then `grpcerror.EnsureStatus`. This path:
1. Maps `engine.NonRetryable` errors to `codes.FailedPrecondition`
2. Maps rejections through i18n catalog to `codes.FailedPrecondition`
3. Maps unknown errors to `codes.Internal` (with server-side logging)
4. Catches leaked non-status errors via `EnsureStatus`

System (Daggerheart) transports use a different pattern:
- Store/domain errors go through `grpcerror.HandleDomainError` (which calls
  `apperrors.HandleError`)
- Precondition failures use inline `status.Error(codes.FailedPrecondition, ...)`
- Write commands go through `deps.ExecuteSystemCommand` / `deps.ExecuteDomainCommand`
  callbacks, which internally use `workflowwrite.ExecuteAndApply` (which uses
  `domainwrite.TransportExecuteAndApply` with `PreserveDomainCodeOnApply: true`)

The practical difference:
- For store-layer `ErrNotFound`: core-game path may leak a raw error;
  system-transport path maps it to a structured gRPC code via `HandleDomainError`
- For domain validation errors: core-game path relies on `EnsureStatus`
  as a catch-all; system path explicitly calls `HandleDomainError`

This means the same `storage.ErrNotFound` will produce different gRPC status
codes depending on whether it's returned from a core-game handler or a system
handler. The core-game campaign read path in
`campaign_read_application.go:27-36` returns raw `Campaign.Get` errors
without any error mapping, relying on the gRPC interceptor or
`EnsureStatus` to fix it up.

**Files showing the pattern divergence:**
- Core-game: `campaigntransport/campaign_mutation_application.go:29-31`
  (raw `c.stores.Campaign.Get` error returned directly)
- System: `damagetransport/handler.go:54-58` (`grpcerror.HandleDomainError(err)`)
- System: `adversarytransport/handler.go:41-44` (`grpcerror.HandleDomainError(err)`)

**Proposal:** Establish a single "load campaign and validate operation"
helper that returns properly mapped gRPC errors. All transport entry points
should call this instead of raw `Campaign.Get` + inline `ValidateCampaignOperation`.
This would eliminate both the inconsistency and the duplicated 2-step
(load + validate) sequence.

---

### F07 -- Campaign load + system guard + operation validation sequence repeated
**Category:** contributor friction
**Severity:** medium

The preamble sequence for system-transport writes:

```go
record, err := h.deps.Campaign.Get(ctx, campaignID)
if err != nil { return ..., grpcerror.HandleDomainError(err) }
if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpCampaignMutate); err != nil {
    return ..., grpcerror.HandleDomainError(err)
}
if err := daggerheartguard.RequireDaggerheartSystem(record, "..."); err != nil {
    return ..., err
}
```

appears in virtually every system transport handler method. Examples:
- `damagetransport/handler.go:53-62`
- `conditiontransport/handler.go:53-62`
- `adversarytransport/handler.go:39-48`
- `adversarytransport/handler_mutations.go` (multiple methods)
- `environmenttransport/handler.go` (multiple methods)
- `gmmovetransport/handler.go` (multiple methods)
- `countdowntransport/handler_create.go`, `handler_update.go`, `handler_delete.go`
- `recoverytransport/handler_rest.go`, `handler_blaze_of_glory.go`

For core-game transports, the equivalent is:
```go
campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
if err != nil { return ..., err }
if err := authz.RequirePolicy(ctx, c.auth, capability, campaignRecord); err != nil {
    return ..., err
}
if err := campaign.ValidateCampaignOperation(campaignRecord.Status, op); err != nil {
    return ..., err
}
```
(seen in `campaign_mutation_application.go:28-37`,
`campaign_status_application.go:20-26, 63-69, 106-114`,
`session_lifecycle_application.go:28-33`, `scene_lifecycle_application.go:29-38`,
`participant_create_application.go:29-36`, `character_create_application.go:37-43`)

**Proposal:** For system transports, create a `guard.RequireDaggerheartCampaign(ctx, store, campaignID, op, message)` that folds all three steps into one call and returns `(CampaignRecord, error)`. For core-game transports, create `authz.RequireCampaignOperation(ctx, deps, capability, store, campaignID, op)`.

---

### F08 -- Session gate guard repeated inline in system transports
**Category:** contributor friction
**Severity:** low

After the campaign/system preamble, many system transport handlers also call:
```go
sessionID, err := validate.RequiredID(grpcmeta.SessionIDFromContext(ctx), "session id")
sceneID := strings.TrimSpace(in.GetSceneId())
if err := daggerheartguard.EnsureNoOpenSessionGate(ctx, h.deps.SessionGate, campaignID, sessionID); err != nil {
    return ..., err
}
```

This 3-line session-ID extraction + gate guard appears in:
- `damagetransport/handler.go:64-71`
- `conditiontransport/handler.go:64-71`
- `adversarytransport/handler_mutations.go` (multiple methods)
- `gmmovetransport/handler.go` (multiple methods)
- `statmodifiertransport/handler.go`
- `environmenttransport/handler.go` (multiple methods)
- `countdowntransport/handler_create.go`, etc.

**Proposal:** Bundle into `guard.RequireSessionContext(ctx, store, campaignID) (sessionID, sceneID, error)`.

---

### F09 -- `Dependencies` struct divergence between core-game and system transports
**Category:** missing best practice
**Severity:** low

Core-game transport packages use a `Deps` struct that carries concrete store
interfaces (`storage.CampaignStore`, `storage.ParticipantStore`, etc.) plus
`domainwrite.WritePath` and `projection.Applier`.

System transport packages use a `Dependencies` struct with:
- Local narrow interface types (`CampaignStore`, `DaggerheartStore`)
- Function callbacks (`ExecuteSystemCommand`, `LoadAdversaryForSession`, `SeedFunc`)

The two styles are both valid but philosophically different:
- Core-game: struct of store interfaces + write path
- System: struct of narrow interfaces + function callbacks for writes

The function-callback approach decouples the transport slice from the
write-path stack but makes the `Dependencies` harder to wire (each callback
must be constructed at the composition root). The narrow-interface approach
follows Go's "define interfaces at point of use" idiom but creates the
duplication documented in F02.

**This is a design observation, not necessarily a problem.** The callback
approach for system transports is reasonable since it lets the parent
daggerheart service own the write-path configuration. But the contract
duplication (F01, F02) is the price paid for this decoupling.

---

### F10 -- `ApplyErrorWithDomainCodePreserve` duplicated across layers
**Category:** anti-pattern
**Severity:** low

The same function exists in three places:
1. `internal/domainwrite/transport.go:151` -- `ApplyErrorWithDomainCodePreserve`
2. `internal/grpcerror/helper.go:37` -- `ApplyErrorWithDomainCodePreserve`
3. `game/handler/write.go:65` -- `ApplyErrorWithCodePreserve` (delegates to #1)

Both #1 and #2 have identical implementations. The `handler` re-export (#3) is
the only path used by core-game transport callers. System transports use #1
via `domainwrite.NormalizeDomainWriteOptionsConfig{PreserveDomainCodeOnApply: true}`.
The `grpcerror` copy (#2) appears unused in production code.

**Proposal:** Delete `grpcerror.ApplyErrorWithDomainCodePreserve` if no
callers remain. Keep the canonical implementation in `domainwrite` and the
re-export in `handler` for core-game convenience.

---

### F11 -- `GameSystemToProto` / `GameSystemFromProto` vs `SystemIDFromGameSystemProto`
**Category:** contributor friction
**Severity:** low

System-ID mapping between proto and domain exists in two overlapping helpers:
- `campaigntransport/mappers.go:144-162` -- `GameSystemToProto`, `GameSystemFromProto`
- `handler/system.go:10-17` -- `SystemIDFromGameSystemProto` (same logic as `GameSystemFromProto` but different return type)
- `handler/system.go:21-26` -- `SystemIDFromCampaignRecord` (different input, same domain)

`GameSystemFromProto` returns `bridge.SystemID("")` on default while
`SystemIDFromGameSystemProto` returns `bridge.SystemIDUnspecified`. These
sentinel values may or may not differ. A contributor adding a new game system
must update 2-3 places.

**Proposal:** Consolidate into a single mapper pair in the `handler` package
(or the `campaigntransport/mappers.go` since it owns the campaign proto
mapping). Make one canonical pair of `ToProto`/`FromProto` and delete the
other.

---

### F12 -- No shared "reload record after command" pattern
**Category:** contributor friction
**Severity:** low

Nearly every write method ends with a reload-from-store step:
```go
updated, err := c.stores.Campaign.Get(ctx, campaignID)
if err != nil { return ..., grpcerror.Internal("load campaign", err) }
return updated, nil
```

This pattern is identical across `EndCampaign`, `ArchiveCampaign`,
`RestoreCampaign`, `UpdateCampaign`, `SetCampaignCover`, etc. Character,
participant, and session transports have equivalent reload steps.

For system transports, the same pattern uses
`grpcerror.HandleDomainError(err)` instead of `grpcerror.Internal(...)`.

**Proposal:** This is minor boilerplate, but could be folded into the
proposed `EntityCommandExecutor` from F04 -- the executor could optionally
accept a reload function and handle the error mapping uniformly.

---

## Refactoring Priority Matrix

| Finding | Severity | Effort | Impact | Priority |
|---------|----------|--------|--------|----------|
| F06 | high | medium | eliminates correctness risk | 1 |
| F01 | medium | low | removes 10 duplicated structs | 2 |
| F02 | medium | low | removes 30+ duplicated interfaces | 3 |
| F04 | medium | medium | reduces 27+ write sites to single calls | 4 |
| F07 | medium | low | removes 15+ preamble sequences | 5 |
| F05 | low | trivial | removes 2 duplicated functions | 6 |
| F10 | low | trivial | removes 1 unused function | 7 |
| F03 | low | medium | removes 18 nil-check methods | 8 |
| F08 | low | low | reduces 10+ guard sequences | 9 |
| F11 | low | low | consolidates 3 mapper pairs | 10 |
| F12 | low | low | optional, subsumable by F04 | 11 |
| F09 | low | n/a | design observation only | -- |
