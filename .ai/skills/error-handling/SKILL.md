---
name: error-handling
description: Structured errors and i18n-friendly messaging workflow
user-invocable: true
---

# Error Handling

Use structured errors (`internal/errors`) to:

1. Enable i18n: error messages can be translated without code changes.
2. Separate concerns: internal messages (for logs) vs external messages (for users).
3. Provide machine-readable codes: clients can handle errors programmatically without parsing strings.

## Creating New Errors

Step 1: Add an error code in `internal/errors/codes.go`:

```go
const (
    // Group by domain (Campaign, Session, Character, etc.)
    CodeMyNewError Code = "MY_DOMAIN_ERROR_NAME"
)
```

Code naming convention: `{DOMAIN}_{SPECIFIC_ERROR}` in SCREAMING_SNAKE_CASE.

Step 2: Map to gRPC code in the `GRPCCode()` switch:

```go
case CodeMyNewError:
    return codes.InvalidArgument // or NotFound, FailedPrecondition, etc.
```

Step 3: Add user-facing message in `internal/errors/i18n/en_us.go`:

```go
"MY_DOMAIN_ERROR_NAME": "Human-readable message with {{.Param}} support",
```

Step 4: Use in domain code:

```go
import apperrors "github.com/louisbranch/fracturing.space/internal/errors"

// Simple error
return apperrors.New(apperrors.CodeMyNewError, "internal: detailed message")

// Error with metadata (for templated user messages)
return apperrors.WithMetadata(
    apperrors.CodeMyNewError,
    "internal: transition from X to Y failed",
    map[string]string{"Param": "value"},
)

// Wrapping another error
return apperrors.Wrap(apperrors.CodeMyNewError, "operation failed", underlyingErr)
```

## When to Create New Error Codes

Create a new code when:

- A distinct failure mode needs specific client handling.
- The error requires translation or localization.
- You need to distinguish this error from similar errors programmatically.

Don't create a new code when:

- The error is purely internal and will never reach clients.
- An existing code already covers the failure mode.
- The error is a transient/retryable condition (use existing `CodeUnknown`).

## Checking Errors

```go
// Check for specific error code
if apperrors.IsCode(err, apperrors.CodeNotFound) {
    // Handle not found
}

// Extract error details
var appErr *apperrors.Error
if errors.As(err, &appErr) {
    log.Printf("Code: %s, Message: %s", appErr.Code, appErr.Message)
}
```

## Error Code Categories

| gRPC Code | Use For |
|-----------|---------|
| `InvalidArgument` | Bad user input, validation failures |
| `NotFound` | Resource doesn't exist |
| `FailedPrecondition` | State doesn't allow operation |
| `Internal` | Unexpected errors (default) |
