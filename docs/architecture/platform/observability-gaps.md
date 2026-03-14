# Observability gaps

The game service has good audit coverage for unary RPCs and OTel tracing
infrastructure. This document captures remaining observability gaps
identified during the game service review.

## Completed

- Streaming audit interceptor coverage (F14.1)
- Domain rejections promoted to audit events (F14.2)
- Projection gap detection emits audit events (F14.3)

## Pending

### Structured logging migration

`log.Printf` remains in 11+ operational paths: audit failures, authz
telemetry, dead-letter detection, session lock blocks, startup phases,
worker loops, and store close errors. These should migrate to `log/slog`
with structured fields for queryability.

### Domain-specific tracing spans

OTel setup exists but no domain-specific spans are created for command
execution, event persistence, or projection apply. A span factory for
domain write operations would enable latency attribution across the
write path.

### Metrics wiring

Two metric constants exist (`game_audit_writes_emitted_total`,
`game_audit_write_errors_total`) but are unused. Wiring these and adding
counters for projection lag, command throughput, and rejection rates
would provide operational health signals.

### Health check expansion

Three capability states are registered at startup. Gaps:

- Campaign service hardcodes `Operational` (never degrades)
- Catalog monitor stops polling after ready (no re-detection)
- No health checks for event store, projection lag, or auth service
