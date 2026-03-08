# GSR Phase 17: MCP Transport Layer

## Summary

The MCP transport layer achieves **comprehensive gRPC parity** with 28 tools and 10+ resources. All handlers follow an identical 7-step delegation pattern with zero business logic leakage. Authentication supports dual-mode (API token + OAuth). Error translation is consistent but could benefit from centralized gRPC status code mapping.

## Findings

### F17.1: gRPC Parity — Complete

**Severity:** style (no action needed)

100% operation coverage: Campaign (6), Character (7), Participant (3), Session (2), Fork (2), Event (1), Daggerheart (6), Context (2 MCP-specific). All mutations and reads have MCP equivalents.

### F17.2: Handler Delegation — Excellent

**Severity:** style (no action needed)

All 50+ handlers follow identical 7-step pattern: invocation context → outgoing gRPC context → validation → gRPC delegation → response transformation → metadata merge → resource notification. Zero business logic in MCP layer.

### F17.3: Error Translation — Good, Could Be Centralized

**Severity:** minor

Handlers use `fmt.Errorf("operation failed: %w", err)` for gRPC errors. Resource handlers explicitly handle `codes.NotFound` and `codes.InvalidArgument`. Other handlers rely on generic wrapping.

**Recommendation:** Create central `translateGRPCErrorToMCP(error) error` function for systematic status code mapping.

### F17.4: Context Propagation — Excellent

**Severity:** style (no action needed)

RequestID, InvocationID, ParticipantID flow through gRPC metadata. `NewOutgoingContext` / `NewOutgoingContextWithContext` handle context enrichment. `CallToolResultWithMetadata` embeds IDs in MCP responses.

### F17.5: Tool Definitions — Complete

**Severity:** style (no action needed)

All 28 tools have schema via `jsonschema` tags. 9 enum types have bidirectional conversion functions. Unknown enums degrade to UNSPECIFIED (gRPC provides validation).

### F17.6: Test Coverage — Good

**Severity:** minor

8 domain test files, 4 service test files. All 28 tools tested for success and error paths. Metadata propagation verified. Edge cases (timeouts, concurrent handlers, large payloads) need expansion.

### F17.7: Authentication — Excellent Parity

**Severity:** style (no action needed)

Transport-level auth (HTTP layer) with API token + OAuth bearer + custom authorizer modes. Equivalent boundary to gRPC interceptors. Constant-time comparison for API tokens. Protected resource metadata endpoint.

## Cross-References

- **Phase 8** (gRPC Transport): MCP delegates to gRPC
- **Phase 10** (Error Handling): Error translation consistency
- **Phase 14** (Observability): Audit via gRPC (not MCP-specific)
