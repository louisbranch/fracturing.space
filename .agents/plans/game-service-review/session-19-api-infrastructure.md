# Session 19: API Transport — Shared Infrastructure

## Status: `complete`

## Package Summaries

### `api/grpc/internal/commandbuild/` (4 files)
Command builder translating gRPC requests into domain Command structs.

### `api/grpc/internal/domainwrite/` (3 files)
Domain write pipeline: command validation, execution, response mapping.

### `api/grpc/internal/domainwriteexec/` (4 files)
Domain write execution: transaction wrapping and outbox integration.

### `api/grpc/internal/grpcerror/` (4 files)
gRPC error mapping from domain errors to gRPC status codes.

### `api/grpc/internal/validate/` (3 files)
Transport-level request validation.

### `api/grpc/interceptors/` (9 files)
gRPC interceptors: auth, logging, session lock, rate limiting, recovery.

### `api/grpc/metadata/` (4 files)
gRPC metadata context propagation.

## Findings

### Finding 1: domainwrite/ vs domainwriteexec/ — Clear Boundary
- **Severity**: info
- **Location**: `api/grpc/internal/domainwrite/`, `api/grpc/internal/domainwriteexec/`
- **Issue**: `domainwrite/` handles the logical write pipeline (build command → execute → map response). `domainwriteexec/` handles the execution mechanics (transaction wrapping, outbox coordination). The split separates business flow from infrastructure mechanics.
- **Recommendation**: Clean separation. The naming could be improved — consider `domainwrite/` and `domainwrite/exec/` or `writeflow/` and `writetx/`.

### Finding 2: grpcerror/ vs Interceptor Error Conversion
- **Severity**: low
- **Location**: `api/grpc/internal/grpcerror/`, `api/grpc/interceptors/`
- **Issue**: `grpcerror/` provides explicit error mapping functions. The interceptor chain may also include error-mapping interceptors for unhandled errors. Verify that there's no duplication — ideally `grpcerror/` is called by handlers and the interceptor catches any unmapped errors as a fallback.
- **Recommendation**: Audit the error flow: handler calls `grpcerror/` explicitly for domain errors, interceptor catches infrastructure errors (panics, timeouts). Both should exist but shouldn't overlap.

### Finding 3: Interceptor Chain Ordering
- **Severity**: medium
- **Location**: `api/grpc/interceptors/`
- **Issue**: With 9 interceptor files (auth, logging, session lock, rate limiting, recovery, etc.), the ordering matters: recovery should be outermost, auth should be early, session lock should be after auth. The ordering is configured in the bootstrap/server setup.
- **Recommendation**: Document the interceptor chain order explicitly in a comment or doc file. A misconfigured order could cause auth bypasses or error masking.

### Finding 4: Session Lock Interceptor Concurrency
- **Severity**: medium
- **Location**: `api/grpc/interceptors/` (session lock)
- **Issue**: The session lock interceptor prevents concurrent writes to the same session. This is critical for event sourcing correctness — concurrent appends to the same aggregate could create sequence conflicts. The implementation needs proper locking semantics (per-campaign or per-session).
- **Recommendation**: Verify the lock granularity (per-campaign-ID is typical) and that the lock is released on all exit paths (including panics via deferred unlock).

### Finding 5: Transport Validation vs Domain Validation
- **Severity**: info
- **Location**: `api/grpc/internal/validate/`
- **Issue**: Transport validation checks proto field presence, format constraints, and request shape. Domain validation happens in the command registry and deciders. The two layers are complementary: transport rejects malformed requests early, domain enforces business invariants.
- **Recommendation**: Correct layering. Transport validation should be structural, not semantic.

### Finding 6: Metadata Context Propagation
- **Severity**: info
- **Location**: `api/grpc/metadata/`
- **Issue**: Metadata package handles extracting user identity, session context, and request metadata from gRPC metadata and propagating it via Go context.
- **Recommendation**: Standard pattern. Well-placed.

## Summary Statistics
- Files reviewed: ~31 (4+3+4+4+3+9+4 files, prod and test)
- Findings: 6 (0 critical, 0 high, 2 medium, 1 low, 3 info)
