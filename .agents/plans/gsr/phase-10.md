# GSR Phase 10: Error Handling & i18n

## Summary

Error handling architecture is **well-structured with clear transport boundaries**. Platform `Error` type used consistently, rejection/error boundary is excellent, i18n catalogs complete for error codes. Key gap: rejection codes (~40 domain-specific codes) are not localized. `PostPersistError` and `NonRetryable` correctly implemented.

## Findings

### F10.1: Rejection Codes Not Localized — Important

**Severity:** important

374 `command.Reject()` uses employ ~40 domain-specific codes (e.g., `PARTICIPANT_ALREADY_JOINED`, `PARTICIPANT_NAME_EMPTY`) that are NOT in i18n catalogs. Error codes (60) are fully translated (en-US + pt-BR), but rejection codes bypass the i18n layer entirely.

**Impact:** Rejections are user-facing validation failures but not translated.

**Recommendation:** Add rejection codes to i18n catalogs. ~40 codes to translate.

### F10.2: Platform Error Type — Consistent

**Severity:** style (no action needed)

`apperrors.New()` and `apperrors.WithMetadata()` used consistently across domain packages. Sentinel errors declared once and reused. 60+ error codes centralized in `platform/errors/codes.go`.

### F10.3: Rejection vs Error Boundary — Excellent

**Severity:** style (no action needed)

Clean semantic separation: `command.Rejection` (with Code + Message) for domain validation failures, `apperrors.Error` for infrastructure errors. Rejections are idempotent domain logic; errors are system failures. Textbook separation.

### F10.4: gRPC `ToGRPCStatus` — Excellent

**Severity:** style (no action needed)

Preserves error code via `ErrorInfo.Reason`, includes domain, stores metadata for client-side templating, includes `LocalizedMessage`.

### F10.5: HTTP vs gRPC Error Mapping — Partial

**Severity:** minor

gRPC mapping is complete (all domain codes → appropriate gRPC codes). HTTP mapping (`shared/httperrors`) is infrastructure-only; game service HTTP endpoints (if any) would lose domain code specificity through `KindInvalidInput` downgrades.

### F10.6: Sentinel Error Comparison — Excellent

**Severity:** style (no action needed)

All sentinel errors compared via `errors.Is()`. No `==` comparisons found in domain code. 50+ correct `errors.Is()` instances.

### F10.7: `PostPersistError` and `NonRetryable` — Good

**Severity:** style (no action needed)

Correctly implemented in engine handler for all three post-persist stages. gRPC transport checks `IsNonRetryable()` and maps to `codes.FailedPrecondition`.

### F10.8: Locale Extraction — Known Gap

**Severity:** minor

Currently hardcoded to `DefaultLocale` with explicit TODO. Intentional deferral pending auth/web metadata propagation alignment.

### F10.9: Error Code Catalog — Complete

**Severity:** style (no action needed)

60 codes with SCREAMING_SNAKE naming, clear domain categories, complete gRPC status mapping. Well-organized.

## Cross-References

- **Phase 4** (Command/Decision): Rejection code conventions
- **Phase 5** (Engine): PostPersistError implementation
- **Phase 8** (gRPC Transport): Error mapping at transport boundary
- **Phase 16** (Web/Admin): HTTP error mapping
