# GSR Phase 8: gRPC Transport Layer

## Summary

The gRPC transport layer demonstrates **strong architectural discipline**. The 172-file flat package is well-justified by consistent naming, comprehensive doc.go, and clear semantic grouping. Authorization is correctly layered (interceptor + per-handler). Error mapping is centralized and complete. One gap: mapper functions lack dedicated unit tests.

## Findings

### F8.1: Flat Layout (172 Files) — Well-Designed

**Severity:** style (no action needed)

File naming conventions (`campaign_*`, `character_*`, `session_*`, etc.) provide clear navigation. Excellent 148-line doc.go documents all services and file organization. All files serve the same aggregate boundary. No sub-packaging needed.

### F8.2: Application/Service Split — Well-Motivated

**Severity:** style (no action needed)

32 `*_application.go` files handle orchestration (command building, domain command execution). Service handlers expose RPC entry points and map proto ↔ domain. Clean separation enables reuse and independent testing.

### F8.3: `Stores` Struct — Not a God Object

**Severity:** style (no action needed)

Stores is a **dependency bundle**, not an orchestrator. Clear semantic grouping (projection stores, infrastructure stores, content stores, runtime). `Validate()` enforces all requirements at startup. No logic in the struct.

### F8.4: Authorization — Correctly Layered

**Severity:** style (no action needed)

Two layers: `SessionLockInterceptor` (protocol-level, blocks mutations during active sessions) and per-handler `requirePolicy()` (domain-level access). Actor resolution centralized via `resolvePolicyActor()`. Audit telemetry on every authorization decision.

### F8.5: Actor Resolution — Consistent

**Severity:** style (no action needed)

Metadata extraction centralized in `metadata/metadata.go`. Interceptors guarantee RequestID/InvocationID presence. ASCII validation prevents log injection. Cascade resolution: ParticipantID → UserID.

### F8.6: `executeAndApplyDomainCommand` — Clean Composition

**Severity:** style (no action needed)

Error mapping explicit: `EnsureStatus()` normalizes domain errors to gRPC status. Error callbacks pluggable via `domainwrite.Options`. `engine.IsNonRetryable()` correctly maps to `codes.FailedPrecondition`.

### F8.7: Mapper Test Coverage — Gap

**Severity:** important

Mapper files (`campaign_mappers.go`, `character_mappers.go`, etc.) lack dedicated unit tests. Exercised indirectly through service tests but proto enum changes could silently miss mapper updates.

**Recommendation:** Add table-driven unit tests for each mapper file, focusing on switch statement exhaustiveness.

### F8.8: No Type Leakage — Clean

**Severity:** style (no action needed)

Domain types never leak into proto responses. Storage records pass through mappers before returning. Proto boundary correctly maintained.

### F8.9: Error Mapping — Comprehensive

**Severity:** style (no action needed)

Centralized in `grpcerror/helper.go`. Covers: validation (InvalidArgument), permission (PermissionDenied), rejection (FailedPrecondition), non-retryable (FailedPrecondition), retryable (Internal), session locked (FailedPrecondition).

## Cross-References

- **Phase 5** (Engine): `NonRetryable` error adoption
- **Phase 10** (Error Handling): gRPC error mapping completeness
- **Phase 11** (Configuration): gRPC server creation in startup
- **Phase 14** (Observability): Audit interceptor
