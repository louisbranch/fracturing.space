# GSR Phase 14: Observability — Audit, Metrics, Tracing

## Summary

Observability has **good audit coverage for unary RPCs** but significant gaps in tracing instrumentation, metrics emission, and operational health checks. Audit events are synchronous with graceful degradation. `log.Printf` remains in 11+ critical operational paths.

## Findings

### F14.1: No Streaming Audit Coverage — Important

**Severity:** important

`ChainStreamInterceptor` has NO audit telemetry — only metadata and internal identity checks. Streaming RPCs (if added) bypass audit entirely.

**Recommendation:** Add streaming interceptor audit telemetry.

### F14.2: Domain Rejections Use `log.Printf` — Important

**Severity:** important

Domain rejections use `OnRejectionInfo` callback that logs via `log.Printf` instead of structured `AuditEvent` emission. Rejections contain CampaignID, CommandType, Code but are not queryable/persistent.

**Recommendation:** Promote domain rejections to audit events.

### F14.3: Projection Gaps/Dead Letters Logged Not Audited — Important

**Severity:** important

Projection gap detection and outbox dead-letter conditions emit `log.Printf` for critical operational state without audit event tracking.

**Recommendation:** Emit audit events for projection anomalies.

### F14.4: No Span Creation in Write Handlers — Minor

**Severity:** minor

OTel setup exists but audit interceptor only reads existing spans. No domain-specific spans for command execution, event persistence, or projection apply.

**Recommendation:** Create span factory for domain write operations.

### F14.5: Metrics Reserved But Not Wired — Minor

**Severity:** minor

Two metric constants exist (`game_audit_writes_emitted_total`, `game_audit_write_errors_total`) but are unused. No observability for audit system health.

### F14.6: Health Checks — Partial

**Severity:** minor

Three capability states registered at startup. Campaign service hardcoded `Operational` (never updates). Catalog monitor stops polling after ready (no re-detection of degradation). No health checks for event store, projection lag, or auth service.

### F14.7: `log.Printf` Instances — 11+ Locations

**Severity:** minor

Unstructured logging in: audit failures, authz telemetry, dead-letter detection, projection gaps, session lock blocks, domain rejections, startup phases, worker loops, store close errors.

**Recommendation:** Event-critical paths should emit audit events or structured error logs.

### F14.8: Audit Context — Sufficient for "Who Did What"

**Severity:** style (no action needed)

Captures TraceID, SpanID, RequestID, CampaignID, SessionID, ActorType, ActorID. Missing for performance debugging: latency, request/response size. Missing for auth audit: read operations don't record authorization decisions.

## Cross-References

- **Phase 5** (Engine): Post-persist error handling
- **Phase 8** (gRPC Transport): Interceptor architecture
- **Phase 11** (Configuration): Status runtime in startup
