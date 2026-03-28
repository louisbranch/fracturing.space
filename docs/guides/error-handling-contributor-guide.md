---
title: "Error handling for contributors"
parent: "Guides"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-03-27"
---

# Error handling for contributors

This guide covers the two error systems, how they propagate through the write
path, and what you need to do when adding a new error.

## Two error layers

### 1. Structured error codes (`platform/errors`)

`platform/errors/codes.go` defines `Code` (a `string` type) with constants
grouped by domain. Each code maps to a gRPC status via `Code.GRPCCode()`.

```go
type Code string

const (
    CodeCampaignNameEmpty Code = "CAMPAIGN_NAME_EMPTY"   // -> InvalidArgument
    CodeActiveSessionExists Code = "ACTIVE_SESSION_EXISTS" // -> FailedPrecondition
    CodeNotFound          Code = "NOT_FOUND"               // -> NotFound
    // ...system-specific codes prefixed by system name
    CodeDaggerheartInvalidDifficulty Code = "DAGGERHEART_INVALID_DIFFICULTY"
)
```

`platform/errors/errors.go` provides constructors:

| Constructor | Use |
|-------------|-----|
| `errors.New(code, msg)` | Simple domain error. |
| `errors.WithMetadata(code, msg, meta)` | Error with i18n template variables. |
| `errors.Wrap(code, msg, cause)` | Wraps an underlying error. |
| `errors.WrapWithMetadata(...)` | Both metadata and cause. |

### 2. Rejection codes (write-path deciders)

`domain/command/decision.go` defines `Rejection{Code, Message}`. Deciders
return `command.Reject(...)` with a stable string code.

Convention:
- **`SCREAMING_SNAKE_CASE`** with a domain prefix: `CAMPAIGN_NAME_EMPTY`,
  `GM_FEAR_OUT_OF_RANGE`.
- Shared codes live in `domain/command/decision.go`
  (`PAYLOAD_DECODE_FAILED`, `COMMAND_TYPE_UNSUPPORTED`).
- Domain-specific codes live in each domain decider and are exported via a
  `RejectionCodes()` function (e.g. `campaign.RejectionCodes()`,
  `session.RejectionCodes()`).

## Write-path propagation

```
command arrives
    |
    v
Decider.Decide(state, cmd, now)
    |-- returns Decision{Events} on accept
    |-- returns Decision{Rejections} on reject
    v
Engine inspects Decision
    |-- accepted: persists events, folds state
    |-- rejected: converts Rejection -> platform/errors.Error
    v
gRPC handler returns error
    v
ErrorConversionUnaryInterceptor (interceptors/error_conversion.go)
    |-- already gRPC status? pass through
    |-- platform/errors.Error? -> HandleError(err, locale)
    |        -> i18n catalog formats user message
    |        -> ToGRPCStatus attaches ErrorInfo + LocalizedMessage
    |-- unknown error? -> codes.Internal with generic message
    v
gRPC response to client
```

The `ErrorConversionUnaryInterceptor` is the single boundary that normalizes
domain errors to gRPC status. Individual handlers should never call
`status.Error()` for domain errors -- just return the `*errors.Error` and let
the interceptor handle locale + details.

## i18n

Localized user-facing messages live in `platform/errors/i18n/`. The catalog
maps `Code` strings to message templates. `HandleError` resolves the caller
locale from gRPC metadata, looks up the template, substitutes `Metadata`
fields, and attaches the result as a `LocalizedMessage` detail on the gRPC
status.

## Adding a new error path

### New structured error code

1. Add the `Code` constant to `platform/errors/codes.go` in the appropriate
   domain group.
2. Add the code to the correct `case` in `GRPCCode()` so it maps to the right
   gRPC status (`InvalidArgument`, `FailedPrecondition`, etc.).
3. Add an i18n message template in `platform/errors/i18n/` for `en-US` (and
   any other supported locales).
4. Use `errors.New(code, internalMsg)` or `errors.WithMetadata(...)` at the
   call site.

### New rejection code (write path)

1. Define the `const` in the appropriate domain decider file
   (e.g. `domain/campaign/decider.go`).
2. Add the code to that domain's `RejectionCodes()` slice so documentation and
   tests stay in sync.
3. Return `command.Reject(command.Rejection{Code: "...", Message: "..."})` from
   the decider.
4. The engine and interceptor handle the rest -- no transport code needed.

### Read-path / lookup errors

For storage lookups, use the `grpcerror` helpers in
`internal/services/game/api/grpc/internal/grpcerror/helper.go`:

- `LookupErrorContext(ctx, err, internalMsg, notFoundMsg)` -- maps
  `storage.ErrNotFound` to `codes.NotFound`, structured domain errors to their
  semantic code, and everything else to `codes.Internal`.
- `OptionalLookupErrorContext(ctx, err, internalMsg)` -- same but treats
  not-found as nil (absent data).
