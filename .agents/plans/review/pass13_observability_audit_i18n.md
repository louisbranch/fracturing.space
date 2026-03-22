# Pass 13: Observability, Audit, and i18n

## Summary

The observability, audit, and i18n subsystems are well-structured and consistently
applied. The audit interceptor chain covers both unary and streaming gRPC calls,
domain rejections are persisted to the audit store, and the i18n catalog system
has a comprehensive tooling pipeline (`i18ncheck`, `i18nstatus`) and full pt-BR
parity. The OTel setup is clean and minimal. The main findings are:

- The `classifyMethodKind` read/write allowlist is incomplete for several newer
  services (Scene, Interaction, AI orchestration), causing all unlisted read
  methods to be classified as "write" in audit events.
- The `AuditInterceptor` creates a new `Emitter` on every request instead of
  reusing a shared instance.
- The `AuditInterceptor` ordering relative to `ErrorConversionUnaryInterceptor`
  means telemetry sees pre-conversion domain errors rather than final gRPC
  status codes for some error paths.
- Auth copy i18n uses inline English fallbacks instead of catalog keys for
  ~40 strings, meaning the auth page has a parallel translation system.
- The OTel provider uses `AlwaysSample()` which is fine for development but
  not documented as requiring a change for production.
- Several metadata context helpers use different storage mechanisms
  (context value vs incoming gRPC metadata) without clear documentation of why.

Severity distribution: 2 correctness risks, 3 missing best practices,
3 anti-patterns, 2 contributor friction.

---

## Findings

### 1. `classifyMethodKind` read allowlist is incomplete for newer services
**Category:** correctness risk
**Files:**
- `internal/services/game/api/grpc/interceptors/telemetry.go:178-210`

The `classifyMethodKind` function uses a static allowlist of read methods,
defaulting to `"write"` for anything not listed. Multiple read-only methods from
newer services are missing from the list:

- `SessionService_ListActiveSessionsForUser_FullMethodName` -- read, not listed
- `SceneService_GetScene_FullMethodName` -- read, not listed
- `SceneService_ListScenes_FullMethodName` -- read, not listed
- `InteractionService_GetInteractionState_FullMethodName` -- read, not listed
- `CharacterService_ListCharacterProfiles_FullMethodName` -- read, not listed
- `CampaignAIService_GetCampaignAIBindingUsage_FullMethodName` -- read, not listed
- `CampaignAIService_GetCampaignAIAuthState_FullMethodName` -- read, not listed
- `InviteService_GetPublicInvite_FullMethodName` -- read, not listed

These will all be classified as `"write"` in durable audit events, making
cross-service telemetry analysis misleading.

**Proposal:** Invert the allowlist: define write methods explicitly and default
to `"read"`, or add the missing entries. Consider a compile-time or startup
validation similar to `ValidateSessionLockPolicyCoverage` to catch drift.

---

### 2. `AuditInterceptor` creates a new `Emitter` per request
**Category:** anti-pattern
**Files:**
- `internal/services/game/api/grpc/interceptors/telemetry.go:60`
- `internal/services/game/api/grpc/interceptors/telemetry.go:121`

Both `AuditInterceptor` and `StreamAuditInterceptor` call
`audit.NewEmitter(policy)` inside the per-request closure. The `Emitter` struct
is cheap (two fields, no allocations beyond the struct), but this is
architecturally wasteful and a friction point for contributors who might assume
it needs per-request state. The emitter could be created once when the
interceptor is constructed and reused.

**Proposal:** Create the emitter once in the interceptor constructor:
```go
func AuditInterceptor(policy audit.Policy) grpc.UnaryServerInterceptor {
    emitter := audit.NewEmitter(policy)
    return func(...) {
        // use emitter
    }
}
```

---

### 3. Audit interceptor observes pre-conversion errors
**Category:** correctness risk
**Files:**
- `internal/services/game/api/grpc/interceptors/doc.go:11-19`
- `internal/services/game/app/bootstrap_transport.go:151-157`

The documented interceptor ordering places telemetry (position 3) outside
error_conversion (position 5), meaning telemetry wraps the handler *before*
error conversion runs. The doc.go comment says the telemetry interceptor
"emits audit events with gRPC status codes" and observes "the final outcome
including error conversion results." However, the actual chain ordering is:

```
metadata -> internal_identity -> telemetry -> session_lock -> error_conversion
```

Since error_conversion is innermost, `telemetry` calls `handler(ctx, req)` which
chains through session_lock and error_conversion. Because Go gRPC chain
interceptors execute in order (outermost wraps innermost), telemetry *does* see
the error after error_conversion runs. The doc statement is therefore correct.
However, if session_lock rejects with a raw `status.Error` (which it does), that
is already a gRPC status and bypasses error_conversion. This is fine but subtle.

No code change needed, but the doc could be clearer about the pass-through path.

**Reclassified:** This is actually correct. Removing this finding. (Kept for
audit trail.)

---

### 4. `Emitter.Emit` has unreachable nil-clock branch
**Category:** anti-pattern (minor)
**Files:**
- `internal/services/game/observability/audit/emitter.go:72-74`

The `NewEmitter` constructor always sets `clock: time.Now`, so the branch
`if e.clock == nil { evt.Timestamp = time.Now().UTC() }` at line 72 is only
reachable if someone constructs an `Emitter` struct literal with a nil clock
(which happens in tests at line 83 of the test file). The defensive fallback
is reasonable but creates contributor confusion about which path is canonical.

**Proposal:** Either remove the nil check (tests should use the constructor) or
document it as an intentional defense-in-depth fallback with a comment.

---

### 5. Auth copy uses 40+ inline English fallbacks instead of catalog keys
**Category:** missing best practice
**Files:**
- `internal/services/web/platform/i18n/auth_copy.go:70-122`

The `Auth()` function uses `localizeWithFallback(loc, key, "English text")`
for every field. The English fallback strings are hardcoded in Go code rather
than being sourced from the catalog. While this works because the catalog
registers the keys and the fallback is only used if the key is missing, it
means:

1. The English strings exist in two places (Go code and the catalog YAML).
2. If a catalog key changes value, the Go fallback may become stale.
3. The `localizeWithFallback` pattern encourages bypassing the catalog.

There is currently no `auth.yaml` or `public.yaml` namespace in the catalog;
the auth copy keys are registered through `i18nmessages` which imports the
catalog package. The auth keys like `"login.heading"` do exist in the
registered catalog (confirmed by the test assertions), so the fallbacks serve
only as belt-and-suspenders safety.

**Proposal:** Consider adding an `auth.yaml` namespace that is the canonical
source, and removing the inline fallbacks (or converting them to a panic/log
at startup if the key is missing).

---

### 6. OTel `AlwaysSample()` sampler not documented for production
**Category:** missing best practice
**Files:**
- `internal/platform/otel/provider.go:55`

The provider always uses `sdktrace.AlwaysSample()`. In production under load,
this sends every span to the collector, which can cause significant overhead.
There is no configuration knob to switch to a probabilistic or rate-limited
sampler.

**Proposal:** Add an env var (e.g., `FRACTURING_SPACE_OTEL_SAMPLE_RATE`) that
defaults to 1.0 (always sample) but allows production to dial it down. Document
the production expectation in `docs/running/`.

---

### 7. OTel propagator only sets `TraceContext`, not `Baggage`
**Category:** missing best practice (minor)
**Files:**
- `internal/platform/otel/provider.go:59`

The global propagator is set to `propagation.TraceContext{}` alone. If any
downstream service or library expects W3C Baggage propagation, it will be
silently dropped. This is fine today since the project does not use baggage,
but it is a common future requirement.

**Proposal:** Use `propagation.NewCompositeTextMapPropagator(
propagation.TraceContext{}, propagation.Baggage{})` as a forward-compatible
default. No functional change for current usage.

---

### 8. Metadata context key storage is inconsistent
**Category:** contributor friction
**Files:**
- `internal/services/game/api/grpc/metadata/metadata.go:56-63` (context values)
- `internal/services/game/api/grpc/metadata/metadata.go:84-101` (gRPC metadata)

`RequestID` and `InvocationID` are stored as context values (via `WithRequestID`
/ `WithInvocationID`) and read via `RequestIDFromContext` /
`InvocationIDFromContext`. But `ParticipantID`, `UserID`, `CampaignID`,
`SessionID`, and `ServiceID` are read directly from `metadata.FromIncomingContext`
on every access.

The difference is intentional: request/invocation IDs are generated/ensured by
the metadata interceptor and need a stable context value, while the others are
pass-through from incoming gRPC metadata. However, there is no documentation
explaining this split, and a contributor adding a new header might follow either
pattern inconsistently.

**Proposal:** Add a doc comment to the package explaining the two-tier context
storage strategy (interceptor-managed values vs pass-through metadata headers).

---

### 9. `StreamAuditInterceptor` cannot capture request-scoped fields
**Category:** contributor friction
**Files:**
- `internal/services/game/api/grpc/interceptors/telemetry.go:87-90`

The stream audit interceptor explicitly documents that `campaignID` and
`sessionID` are not available because the interceptor does not have access to
the request message. The audit event is emitted with empty scope fields.

This is a known limitation documented in the code comment, but it means stream
audit events lack the routing dimensions that unary events have. For the single
current streaming RPC (`SubscribeCampaignUpdates`), the campaign ID is part of
the request and could theoretically be extracted if the interceptor wrapped the
stream to intercept the first `RecvMsg`.

**Proposal:** For now, this is acceptable since there is only one streaming RPC.
If more streaming RPCs are added, consider a `wrappedServerStream` that peeks
at the first message to extract scope fields.

---

### 10. Audit event `Severity` all non-OK errors as ERROR regardless of code
**Category:** anti-pattern
**Files:**
- `internal/services/game/api/grpc/interceptors/telemetry.go:38-45`

Both the unary and stream audit interceptors set severity to `ERROR` for any
non-nil handler error, regardless of the gRPC status code. A `NotFound` or
`InvalidArgument` error represents a normal request rejection (client error),
not an operational error. This inflates the error severity in audit records
and makes it harder to distinguish actual system failures from expected
rejections.

**Proposal:** Map client error codes (NotFound, InvalidArgument, PermissionDenied,
FailedPrecondition, etc.) to `SeverityWarn` and reserve `SeverityError` for
server-side failures (Internal, Unavailable, DataLoss, etc.).

---

## Non-Findings (Confirmed Correct)

1. **i18n key parity between en-US and pt-BR:** The `errors.yaml`, `game.yaml`,
   `notifications.yaml`, `core.yaml`, and `admin.yaml` files have 1:1 key
   coverage between locales. The `i18ncheck` tool validates placeholder parity.

2. **Audit policy explicit enablement:** The `audit.Policy` type uses explicit
   enable/disable rather than nil-store inference, which is documented and tested.

3. **Interceptor ordering is correct:** The chain in `bootstrap_transport.go`
   matches the documented ordering in `interceptors/doc.go`.

4. **OTel resource construction avoids schema URL conflicts:** Uses
   `resource.New` with `resource.WithAttributes` rather than `resource.Merge`,
   which is the correct pattern per project memory.

5. **Domain rejection audit events:** The `domainwrite/transport.go` package
   correctly emits `telemetry.domain.rejection` audit events when audit is
   enabled, with structured attributes including command type and rejection code.

6. **Projection gap detection:** The applier watermark correctly emits
   `telemetry.projection.gap_detected` audit events when sequence gaps are
   found during projection.

7. **Game content i18n keys:** Game content strings (participant names, session
   names, readiness messages) correctly use i18n keys with English fallback
   constants, matching the project convention.
