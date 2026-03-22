# Pass 3: Error Handling Architecture

## Summary

The game service uses **four distinct error systems** that coexist across the
write path: (1) plain Go sentinels (`errors.New`), (2) structured domain errors
(`apperrors.Error` with codes and i18n), (3) domain rejections (string codes on
`command.Rejection`), and (4) gRPC statuses. Each system has a clear role, but
they overlap in practice, creating inconsistencies in how errors are
constructed, propagated, and localized. The most actionable findings are the
hardcoded `"en-US"` locale in the rejection-to-gRPC path (finding 1), the
duplicated `ApplyErrorWithDomainCodePreserve` helper (finding 4), and the
absence of system-level rejection code uniqueness validation (finding 6).

---

## Error System Taxonomy

| System | Construction | Transport conversion | i18n support |
|---|---|---|---|
| **Sentinels** (`errors.New`) | `var Err... = errors.New(...)` | Interceptor catches, maps to `codes.Internal` | None |
| **Structured** (`apperrors.Error`) | `apperrors.New(code, msg)` | `HandleError` -> `ToGRPCStatus` with `errdetails` | Yes, via `i18n.Catalog` |
| **Rejections** (`command.Rejection`) | `command.Reject(Rejection{Code, Message})` | `RejectErr` callback in `domainwrite.Options` | Partial (see finding 1) |
| **PostPersist** (`PostPersistError`) | `newPostPersistError(stage, ...)` | `ExecuteErr` callback; `IsNonRetryable` -> `FailedPrecondition` | None |

### Error flow through layers

```
Decider -> engine.Handler -> domainwrite.ExecuteAndApply -> transport.go NormalizeDomainWriteOptions -> gRPC response
                                                                    ^
                                                                    |
                                                           interceptor/error_conversion.go (fallback)
```

---

## Findings

### 1. Hardcoded `"en-US"` locale in rejection-to-gRPC conversion

**Category:** correctness risk, missing best practice

**File:** `internal/services/game/api/grpc/internal/domainwrite/transport.go:139-140`

```go
options.RejectErr = func(code, message string) error {
    cat := errori18n.GetCatalog("en-US")
```

The locale is hardcoded rather than derived from the request context (e.g.,
`Accept-Language` gRPC metadata). The `grpc/metadata` package has no locale
extraction helper, and no interceptor populates a locale onto the context. This
means:

- All rejection messages are always in `en-US`, even if the i18n catalog supports
  other locales.
- The `grpcerror.HandleDomainError` path (in `helper.go:20`) also hardcodes
  `apperrors.DefaultLocale` (`"en-US"`), making the problem systemic rather than
  isolated to one call site.
- The `ErrorConversionUnaryInterceptor` calls `grpcerror.HandleDomainError`
  which again uses `DefaultLocale`.

**Proposal:** Add a `locale` key to gRPC metadata extraction
(`grpc/metadata/metadata.go`), populate it from incoming metadata in an
interceptor (or from session/campaign locale), and thread it through
`NormalizeDomainWriteOptions` and `HandleError`. This requires a small
interface change to the `RejectErr` callback or a context-scoped locale.

---

### 2. Inconsistent use of `apperrors` vs plain `errors.New` across domain packages

**Category:** anti-pattern, contributor friction

Plain `errors.New` sentinels and structured `apperrors.New` errors coexist in
the domain layer with no clear rule for which to use. Some packages use
`apperrors` consistently, others use plain sentinels:

**Using `apperrors.New` (structured, with codes):**
- `domain/campaign/errors.go` -- all 4 errors use `apperrors.New`
- `domain/campaign/policy.go:37` -- `ErrCampaignStatusDisallowsOperation`
- `domain/action/errors.go` -- `ErrOutcomeAlreadyApplied`
- `domain/fork/fork.go:14` -- `ErrEmptyCampaignID`
- `domain/systems/daggerheart/profile/profile.go` -- 10 validation errors
- `domain/systems/daggerheart/mechanics/character_state.go` -- 3 resource errors
- `domain/systems/daggerheart/mechanics/rest.go` -- `ErrInvalidRestSequence`
- `domain/systems/daggerheart/domain/types.go` -- 2 dice/difficulty errors

**Using plain `errors.New` (no code, no i18n):**
- `domain/engine/handler.go:16-45` -- 10 sentinel errors
- `domain/command/registry.go:22-36` -- 7 sentinel errors
- `domain/command/decision.go:10` -- `ErrDecisionEmpty`
- `domain/event/registry.go:22-44` -- 10 sentinel errors
- `domain/ids/errors.go:10` -- `ErrCampaignIDRequired`
- `domain/module/registry.go:19-31` -- 6 sentinel errors
- `domain/replay/replay.go:22-31` -- 4 sentinel errors
- `domain/systems/registry_bridge.go:243-250` -- 5 sentinel errors
- `domain/systems/adapter_registry.go:40-50` -- 5 sentinel errors

**The distinction matters** because the `ErrorConversionUnaryInterceptor` at
`interceptors/error_conversion.go:25` attempts `grpcerror.HandleDomainError`,
which calls `apperrors.HandleError`. For plain `errors.New` sentinels,
`errors.As(err, &appErr)` fails, and the fallback returns
`codes.Internal` + `"an unexpected error occurred"`. This means:

- A command validation failure like `ErrTypeRequired` (plain sentinel) leaks to
  gRPC as `codes.Internal` rather than `codes.InvalidArgument`.
- The same logical category of error (e.g., `CAMPAIGN_NAME_EMPTY`) gets
  `codes.InvalidArgument` when raised via `apperrors` but `codes.Internal` when
  raised via a plain sentinel at a different code path.

**Most of the plain sentinels in engine/command/event packages are
infrastructure errors** (missing registry, nil journal) that should indeed be
`codes.Internal`. But `ErrPayloadInvalid`, `ErrTypeUnknown`, `ErrActorTypeInvalid`,
`ErrActorIDRequired` in `command/registry.go` are user-input validation errors
that reach the transport layer and get mapped to `codes.Internal` instead of
`codes.InvalidArgument`.

**Proposal:** Either:
- Convert user-facing validation sentinels in `command/registry.go` and
  `event/registry.go` to use `apperrors.New` with appropriate codes, or
- Add those sentinel errors to the `GRPCCode()` switch in the interceptor
  mapping (less desirable, creates a second mapping table).

---

### 3. Parallel error systems for the same concept: rejection codes vs `apperrors.Code`

**Category:** anti-pattern, contributor friction

Domain rejections (`command.Rejection.Code` as plain strings) and platform
errors (`apperrors.Code` typed strings) are independent namespaces that overlap
semantically:

| Concept | Rejection code (decider) | apperrors.Code (platform) |
|---|---|---|
| Campaign name empty | `CAMPAIGN_NAME_EMPTY` | `CodeCampaignNameEmpty` = `"CAMPAIGN_NAME_EMPTY"` |
| Outcome already applied | `OUTCOME_ALREADY_APPLIED` | `CodeOutcomeAlreadyApplied` = `"OUTCOME_ALREADY_APPLIED"` |
| Active session exists | `SESSION_READINESS_ACTIVE_SESSION_EXISTS` | `CodeActiveSessionExists` = `"ACTIVE_SESSION_EXISTS"` |

The string values sometimes match and sometimes don't. The rejection i18n path
in `transport.go:140` does a catalog lookup by rejection code, which works when
the rejection code string matches an i18n catalog key. But the `apperrors.Code`
constants are the canonical i18n keys (defined in `codes.go`), while rejection
codes are independently defined in each decider file.

There is **no validation** that rejection code strings have corresponding i18n
catalog entries. When a rejection code has no catalog entry, the fallback at
`transport.go:141-142` uses the raw domain message, which is hardcoded English.

**Proposal:** Either:
- Unify by having rejection codes reference `apperrors.Code` constants, or
- Add startup validation that all exported `RejectionCodes()` have corresponding
  entries in the error i18n catalog.

---

### 4. Duplicated `ApplyErrorWithDomainCodePreserve` function

**Category:** contributor friction

The same function exists in two places with identical implementations:

1. `api/grpc/internal/grpcerror/helper.go:37-44` -- `ApplyErrorWithDomainCodePreserve`
2. `api/grpc/internal/domainwrite/transport.go:151-158` -- `ApplyErrorWithDomainCodePreserve`

Both are called from different consumers:
- `domainwrite/transport.go:131` calls the local copy.
- `api/grpc/game/handler/write.go:66` calls the `domainwrite` copy.

The `grpcerror` package has an AST-based centralization test
(`helper_centralization_test.go`) that checks for duplicate status helpers, but
it does not catch this duplication because the banned function names
(`ensureGRPCStatus`, `normalizeGRPCDefaults`) don't include
`ApplyErrorWithDomainCodePreserve`.

**Proposal:** Delete one copy and have all callers use the canonical one in
`grpcerror`. Add the function name to the centralization test's disallowed set.

---

### 5. `PAYLOAD_DECODE_FAILED` and `COMMAND_TYPE_UNSUPPORTED` rejection codes are intentionally shared but scattered

**Category:** missing best practice

These two codes are defined as constants in `command/decision.go:19,23` and
then re-aliased in every decider that uses them:

- `campaign/decider.go:44-45` re-declares as local `const`
- `scene/decider_shared.go:26` and multiple `scene/decider_*.go` files use the
  `command.RejectionCodePayloadDecodeFailed` constant directly
- `daggerheart/internal/decider/decider.go:79-80` re-declares as local `const`
- `engine/core_command_router.go:65` uses the string literal `"COMMAND_TYPE_UNSUPPORTED"` directly
- `domain/decide/flow.go:35,86,143,202` uses the string literal `"PAYLOAD_DECODE_FAILED"` directly

The string literals in `core_command_router.go:65` and `decide/flow.go` bypass
the canonical constants entirely.

**Proposal:** Replace all string literals with the `command.RejectionCode*`
constants. This is a small grep-and-replace.

---

### 6. No startup validation for system-level rejection code uniqueness

**Category:** correctness risk

Core domain rejection codes are validated for uniqueness at startup via
`ValidateCoreRejectionCodeUniqueness()` in
`engine/registries_validation_core.go:183-203`. This checks all
`CoreDomain.RejectionCodes()` returns.

However, **system-level rejection codes** (e.g., Daggerheart's 25+ rejection
codes) are **not checked for uniqueness** against core codes or against other
system modules. The daggerheart decider's `PAYLOAD_DECODE_FAILED` and
`COMMAND_TYPE_UNSUPPORTED` intentionally duplicate the core shared codes, but
there is no validation that other overlaps are intentional.

Today's `ValidateCoreRejectionCodeUniqueness` only iterates `CoreDomains()`.
System modules are registered through `module.Registry`, which has no
`RejectionCodes()` interface.

**Proposal:** Add a `RejectionCodeExporter` interface to `module.Module` (optional)
and extend the startup validation to check for collisions between core and
system rejection codes, excluding the two intentionally shared codes.

---

### 7. Error context loss in engine handler infrastructure errors

**Category:** anti-pattern

Several infrastructure errors in the engine handler are returned without
wrapping context:

- `handler.go:279` -- `return command.Command{}, ErrCommandRegistryRequired`
  (no campaign ID, no command type)
- `handler.go:287` -- `return command.Decision{}, ErrGateStateLoaderRequired`
  (no campaign context)
- `handler.go:342` -- `return command.Decision{}, ErrDeciderRequired` (no context)

These sentinel errors carry no request context, making it harder to diagnose
which request triggered the error. In contrast, `PostPersistError` correctly
captures `CampaignID` and `LastSeq`.

These are startup-safety errors (caught by `NewHandler`), so they should
rarely occur at runtime. The test-path flexibility comment acknowledges this.
However, if triggered at runtime due to a misconfigured handler, the error
message gives no diagnostic context.

**Proposal:** Low priority. Consider wrapping with `fmt.Errorf` to add campaign
ID context when returning these errors from the request path (not from
`NewHandler` validation).

---

### 8. Validator errors in daggerheart use plain `errors.New` -- context invisible to transport

**Category:** anti-pattern, missing best practice

The `domain/systems/daggerheart/internal/validator/` package contains ~100+
`errors.New(...)` calls for event payload validation. These errors surface
when the event registry's `ValidateForAppend` calls the validator. If
validation fails:

1. `handler.go:393` returns the error from `h.Events.ValidateForAppend(evt)`.
2. `domainwrite` wraps it via `ExecuteErr` callback.
3. In the gRPC transport, `NormalizeDomainWriteOptions` maps it to
   `grpcInternal(message, err)`, returning `codes.Internal`.

These validation errors are **programming errors** (a decider produced an
invalid event), not user-facing errors, so `codes.Internal` is arguably
correct. But the error messages are English strings that could end up in logs
without structured codes, making them harder to search and triage.

**Proposal:** Low priority. Consider using a structured validator error type
with a code for log searchability, or accept that these are internal-only.

---

### 9. `HandleDomainError` swallows non-`apperrors` errors silently

**Category:** correctness risk

In `platform/errors/grpc.go:33-34`:

```go
// Unknown error - return internal with generic message
return status.Error(codes.Internal, "an unexpected error occurred")
```

When a non-`apperrors.Error` error reaches `HandleDomainError`, the original
error message is **completely discarded**. The caller gets "an unexpected error
occurred" with no error details, and the original error is not logged at this
level. This is intentional security (not leaking internals), but creates a
debugging gap when the error is not logged upstream.

The `grpcerror.Internal()` helper in `grpcerror/helper.go:14` does log before
sanitizing, but `HandleDomainError` at `helper.go:20` does not log -- it
delegates to `apperrors.HandleError` which silently drops unknown errors.

The `ErrorConversionUnaryInterceptor` at `interceptors/error_conversion.go:25`
calls `grpcerror.HandleDomainError(err)` for any non-gRPC-status error. For
plain `errors.New` sentinels, this drops the error message without logging.

**Proposal:** Add `slog.Error` logging in `HandleDomainError` (or in the
interceptor) before converting unknown errors to generic internal status. This
ensures no error is silently swallowed.

---

### 10. `grpcerror.HandleDomainError` always uses `DefaultLocale`

**Category:** anti-pattern

`grpcerror/helper.go:20`:
```go
func HandleDomainError(err error) error {
    return apperrors.HandleError(err, apperrors.DefaultLocale)
}
```

This function is called by the `ErrorConversionUnaryInterceptor` and
`ErrorConversionStreamInterceptor`, meaning ALL domain errors flowing through
the interceptor are localized to `en-US` regardless of the client's locale
preference. This is the same root issue as finding 1, but at a different code
path -- the interceptor fallback.

**Proposal:** Same as finding 1. Thread locale through context.

---

### 11. Error `Is()` on `apperrors.Error` matches by code, not identity

**Category:** correctness risk (minor)

`platform/errors/errors.go:30-35`:
```go
func (e *Error) Is(target error) bool {
    if t, ok := target.(*Error); ok {
        return e.Code == t.Code
    }
    return false
}
```

This means `errors.Is(err1, err2)` returns `true` when two different error
instances share the same code, even if messages/causes differ. This is
intentional for code-based matching, but it breaks Go's conventional `errors.Is`
semantics where identity (not equality) is expected.

The `TestErrorIs` test at `errors_test.go:63-76` documents this behavior.
`errors.Is(err1, err2)` returns `true` for two different `*Error` values with
the same code. This can cause unexpected matches in `errors.Is` chains when
wrapping multiple `apperrors.Error` values.

**Proposal:** Document this deviation from standard `errors.Is` semantics more
prominently (e.g., in the `Error` type doc comment). Consider whether `errors.Is`
should check identity instead, with a separate `IsCode()` for code-based matching
(which already exists as `apperrors.IsCode`).

---

### 12. Rejection message content is hardcoded English across all deciders

**Category:** missing best practice

All `command.Rejection.Message` values across every decider are English strings:

- `campaign/decider.go` -- `"campaign has already been created"`
- `scene/decider_player_phase.go` -- `"decode %s payload: %v"`
- `session/decider_lifecycle.go` -- `"session already started"`
- `participant/decider_join.go` -- `"participant already joined"`
- `daggerheart/internal/decider/state_conditions.go` -- `"fear_spent must be greater than zero"`

The `RejectErr` callback in `transport.go:139-145` attempts i18n by looking up
the rejection code in the catalog. If found, the localized message replaces the
English one. If not found, the raw English message is used as the gRPC status
message.

This design means:
- Rejection messages are user-facing (they become gRPC status messages).
- i18n coverage is opt-in per rejection code -- uncovered codes leak English.
- There is no compile-time or startup-time check for i18n coverage of rejection codes.

**Proposal:** Add an i18n catalog entry for every rejection code exported by
`RejectionCodes()`. Add a startup or test-time validation that all exported
rejection codes have catalog entries (similar to
`ValidateCoreRejectionCodeUniqueness` but for i18n coverage).

---

## Priority Summary

| # | Finding | Category | Priority |
|---|---------|----------|----------|
| 1 | Hardcoded `"en-US"` in rejection path | correctness risk | High |
| 9 | `HandleDomainError` swallows unknown errors without logging | correctness risk | High |
| 10 | `HandleDomainError` always uses `DefaultLocale` | anti-pattern | High (same root as 1) |
| 2 | Inconsistent `apperrors` vs `errors.New` | anti-pattern | Medium |
| 3 | Parallel rejection codes vs `apperrors.Code` | anti-pattern | Medium |
| 4 | Duplicated `ApplyErrorWithDomainCodePreserve` | contributor friction | Medium |
| 5 | Scattered string literals for shared rejection codes | missing best practice | Medium |
| 6 | No system rejection code uniqueness validation | correctness risk | Medium |
| 12 | No i18n coverage validation for rejection codes | missing best practice | Medium |
| 7 | Infrastructure errors lack request context | anti-pattern | Low |
| 8 | Validator `errors.New` invisible to transport | anti-pattern | Low |
| 11 | `Error.Is()` matches by code not identity | correctness risk (minor) | Low |
