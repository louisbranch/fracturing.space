# Pass 14: Go Idioms, Naming, and Documentation

## Summary

Overall the codebase demonstrates strong Go conventions: interfaces are defined at
consumption points, doc comments on exported types routinely explain "why" rather
than restating the signature, and domain boundaries are expressed through clear
package names. The review surfaced a small cluster of recurring patterns worth
addressing.

**Critical findings:** None.

**High-value improvements:**

- One exported method (`Handler.Execute`) is missing its doc comment.
- Receiver name inconsistency in `aggregate.Folder` (`a` vs expected `f`).
- Two production adapter stubs carry unused context/event/payload parameters
  that should be documented or reconsidered.
- Architecture doc (grpc-write-path.md) references function names that no longer
  exist in the codebase.
- `log.Printf` vs `slog` inconsistency in handler/profile_snapshot.go.

---

## Findings

### 1. Missing doc comment on exported method `Handler.Execute`

**Category:** missing best practice
**File:** `internal/services/game/domain/engine/handler.go:172`
**Details:** `Handler.Execute` is the primary entrypoint for all domain command
execution. It is the only exported method on `Handler` and it lacks a doc
comment, while every other exported symbol in the file is documented.
**Proposal:** Add a doc comment explaining that Execute orchestrates the full
command pipeline (validate, gate, load, decide, append, fold, checkpoint) and
returns both the decision and post-fold state for read-after-write flows.

---

### 2. Unused `context.Context` parameter on `evaluateGate`

**Category:** anti-pattern (confirmed)
**File:** `internal/services/game/domain/engine/handler.go:315`
**Details:** `evaluateGate` receives `_ context.Context` but never uses it. The
context is already captured by the `check` closure that calls the gate state
loaders. The parameter exists because `evaluateSessionGate` and
`evaluateSceneGate` pass `ctx` through the closure, not through `evaluateGate`.
**Proposal:** Remove the context parameter from `evaluateGate`. The closures
already capture `ctx` from their enclosing scope. This eliminates a false signal
that the function performs I/O directly.

---

### 3. Unused parameters on adapter stub methods

**Category:** anti-pattern
**File:** `internal/services/game/domain/systems/daggerheart/internal/adapter/handle_progression.go:147-152`
**Details:** `HandleConsumableUsed` and `HandleConsumableAcquired` discard all
three parameters (`_ context.Context`, `_ event.Event`, `_ payload.*`). These
are registered adapter handlers that do nothing. If they are intentional
no-ops (e.g. projection is not needed yet for these events), the convention
should be documented at the call site or in a code comment.
**Proposal:** Add a brief comment explaining why these are intentional no-ops
(e.g., "Consumable events are tracked in the journal for replay but have no
projection-side materialization yet"). Alternatively, if they will never need
projection, consider whether they should use `IntentReplayOnly` or
`IntentAuditOnly` intent so the adapter does not need a handler at all.

---

### 4. Receiver name inconsistency: `a *Folder` in aggregate package

**Category:** contributor friction
**File:** `internal/services/game/domain/aggregate/folder.go:36,53,67`
**Details:** The `Folder` type uses receiver name `a` (probably inherited from
when this was called `Applier`). Go convention recommends a consistent
one-letter abbreviation of the type name. Other types in the same package
layer use idiomatic names (`r` for `Registry`, `g` for `DecisionGate`,
`h` for `Handler`, `d` for `CoreDecider`).
**Proposal:** Rename the receiver from `a` to `f` on all `Folder` methods in
`aggregate/folder.go`. The doc comment already explains the "Folder not Applier"
naming choice, so the leftover `a` receiver is a vestige of the old name.

---

### 5. `handler` package name: "too generic" assessment

**Category:** contributor friction (low severity)
**File:** `internal/services/game/api/grpc/game/handler/doc.go`
**Details:** The package is named `handler` and sits at
`.../api/grpc/game/handler/`. In isolation the name is generic, but in context
it is well-scoped: it is an `internal` package under the game gRPC tree, and
every import site already carries the `game/handler` path qualifier. No other
package in the service is named `handler`, so there is no ambiguity. The doc.go
clearly describes its role as "shared handler utilities."
**Conclusion:** The name is acceptable. A rename (e.g., `transportutil`,
`handlerkit`) would not materially improve clarity and would churn many import
sites. **No action recommended.**

---

### 6. Naming stutter assessment: `command.Command`, `event.Event`, `module.Module`

**Category:** contributor friction (accepted trade-off)
**Files:**
- `internal/services/game/domain/command/registry.go` — `Command` struct
- `internal/services/game/domain/event/registry.go` — `Event` struct
- `internal/services/game/domain/module/registry.go` — `Module` interface
**Details:** These create stutter at call sites: `command.Command`,
`event.Event`, `module.Module`. Go convention discourages this, but in each
case the package name matches the domain concept and there is no natural
alternative name for the primary type. Renaming the packages (e.g., `cmd`,
`evt`, `mod`) would lose domain clarity. Renaming the types (e.g.,
`command.Envelope`, `event.Record`, `module.System`) would add indirection
that obscures intent.
**Conclusion:** The stutter is an accepted trade-off. The codebase is
consistent about it, and the domain language doc uses these names canonically.
**No action recommended.**

---

### 7. Architecture doc uses stale function names

**Category:** contributor friction (documentation drift)
**File:** `docs/architecture/foundations/grpc-write-path.md`
**Details:** The doc references function names that no longer exist in code:

| Doc reference | Actual codebase name |
|---|---|
| `executeAndApplyDomainCommand` | `handler.ExecuteAndApplyDomainCommand` (exported) |
| `executeDomainCommandWithoutInlineApply` | `handler.ExecuteWithoutInlineApply` (exported) |
| `normalizeGRPCDefaults` | `domainwrite.NormalizeDomainWriteOptions` |
| `ensureGRPCStatus` | `grpcerror.EnsureStatus` |
| `handleDomainError` (described as a boundary) | `grpcerror.HandleDomainError` |

The document's sequence diagram and prose description are still accurate in
flow, but every concrete function reference is outdated. A contributor reading
the doc and searching the codebase would find zero matches for most names.
**Proposal:** Update all function references to match current exported names.
Update the `Options` code block to include `InlineApplyEnabled`, `ShouldApply`,
and `OnRejection` fields which were added after the doc was last reviewed.

---

### 8. Mixed `log.Printf` and `slog` usage in production code

**Category:** anti-pattern
**File:** `internal/services/game/api/grpc/game/handler/profile_snapshot.go:45,121`
**Details:** Two `log.Printf` calls use the stdlib `log` package while the rest
of the game service production code consistently uses `log/slog` for structured
logging. These are the only non-test `log.Printf` occurrences in the game
service.
**Proposal:** Migrate both calls to `slog.Info` / `slog.Error` with structured
key-value pairs for `user_id`, `participant_id`, etc. This aligns with the
existing convention and makes these log lines queryable in structured logging
backends.

---

### 9. doc.go quality: most are good, a few are thin

**Category:** contributor friction (low severity)
**Files with strong "why" context:**
- `domain/engine/doc.go` — explains runtime seam, session-start exception
- `domain/aggregate/doc.go` — explains `any` map trade-off
- `domain/module/doc.go` — explains module vs systems boundary
- `domain/command/doc.go` — explains registry purpose
- `domain/event/doc.go` — explains event contract stability
- `projection/doc.go` — explains handler ordering invariants
- `storage/doc.go` — explains covered domains, error types
- `core/doc.go` — explains design philosophy with concrete examples
- `platform/otel/doc.go` — explains env vars and opt-in behavior
- `platform/id/doc.go` — explains encoding choice and output properties

**Files that are thin but adequate:**
- `platform/grpc/doc.go` — "shared gRPC helpers" (one line, no why context)
- `campaigntransport/doc.go` — "owns the campaign gRPC service" (factual but no why)

**Assessment:** The doc.go coverage is well above average for a Go project.
The thin entries are in straightforward utility or transport packages where
the package path already communicates purpose. **No action recommended** for
the thin entries, though `platform/grpc/doc.go` could benefit from noting
`DefaultClientDialOptions` and the stats-handler-based OTel integration.

---

### 10. Game-system authoring guide completeness

**Category:** missing best practice (low severity)
**Files:**
- `docs/architecture/systems/adding-a-game-system.md`
- `docs/guides/adding-command-event-system.md`
- `docs/audience/system-developers.md`
**Details:** The authoring documentation is comprehensive and well-organized.
It covers module/metadata/adapter responsibilities, manifest wiring, storage
contracts, startup validation, and verification commands. One gap: the guide
does not mention the `TypedFolder` and `TypedDecider` generics helpers in
`domain/module/typed.go`, which are the recommended way for system authors to
avoid manual `any` → typed assertions. A system author reading only the docs
would likely write manual type switches before discovering these helpers.
**Proposal:** Add a short section in "Step 2: Implement the Module" pointing to
`TypedFolder[S]` and `TypedDecider[S]` as the recommended wrappers for fold and
decide functions, with a one-line example.

---

### 11. `Applier` receiver name `a` in projection package

**Category:** contributor friction (consistent within package, acceptable)
**File:** `internal/services/game/projection/applier.go` and related files
**Details:** The projection `Applier` struct uses `a` as its receiver name
consistently across all ~30 methods in the projection package. While `a` is
not the idiomatic first-letter abbreviation of `Applier` (which would also be
`a`), it is consistently applied. This is actually fine since `a` for `Applier`
is the correct first-letter convention.
**Conclusion:** No action needed. The receiver name follows Go convention.

---

### 12. `ReplayGateStateLoader.LoadSession` ignores sessionID parameter

**Category:** anti-pattern (minor)
**File:** `internal/services/game/domain/engine/loader.go:139`
**Details:** `LoadSession(ctx, campaignID, _ string)` discards the sessionID
parameter because it replays the full campaign aggregate and extracts the
session sub-state. The interface `GateStateLoader` requires the sessionID
parameter (which is correct for the contract), but the implementation ignores
it. This is documented indirectly by the comment "narrowed to session only
because gate policy is always session-scoped."
**Proposal:** Add a brief inline comment: `// sessionID is unused because the
full aggregate replay includes session state; the parameter satisfies the
GateStateLoader interface contract.`

---

## Summary Table

| # | Category | Severity | Action |
|---|---|---|---|
| 1 | Missing best practice | Medium | Add doc comment to `Handler.Execute` |
| 2 | Anti-pattern | Low | Remove unused `_ context.Context` from `evaluateGate` |
| 3 | Anti-pattern | Low | Document or reconsider no-op adapter stubs |
| 4 | Contributor friction | Low | Rename receiver `a` to `f` on `Folder` |
| 5 | Contributor friction | None | No action (handler name is acceptable) |
| 6 | Contributor friction | None | No action (naming stutter is accepted trade-off) |
| 7 | Contributor friction | Medium | Update stale function names in grpc-write-path.md |
| 8 | Anti-pattern | Low | Migrate `log.Printf` to `slog` in profile_snapshot.go |
| 9 | Contributor friction | None | No action (doc.go quality is good) |
| 10 | Missing best practice | Low | Mention TypedFolder/TypedDecider in authoring guide |
| 11 | Contributor friction | None | No action (receiver name is correct) |
| 12 | Anti-pattern | Low | Add clarifying comment on ignored sessionID |
