# Session 23: Daggerheart API — Service Layer

## Status: `complete`

## Package Summaries

### `api/grpc/systems/daggerheart/` root (~60 files)
Daggerheart system service: RPC handler registration, workflow handlers, and transport-level logic for all Daggerheart operations.

### `api/grpc/systems/daggerheart/guard/` (2 files)
System guard ensuring the campaign's game system matches before allowing Daggerheart operations.

### `api/grpc/systems/daggerheart/gameplaystores/` (3 files)
Gameplay-specific store wiring for Daggerheart transport handlers.

## Findings

### Finding 1: 60+ Files at Package Root — Too Many
- **Severity**: high
- **Location**: `api/grpc/systems/daggerheart/`
- **Issue**: Similar to the domain-level daggerheart package, the API transport has 60+ files at root. This includes all Daggerheart RPC handlers, test files, and workflow logic. Contributors must navigate a flat list of 60+ files to find the relevant handler.
- **Recommendation**: The package already has 15+ sub-packages for specific transports. Move remaining root-level handlers into sub-packages. Root should contain only: service registration (`service.go`), shared types, and the guard.

### Finding 2: service.go as God-Registration File
- **Severity**: medium
- **Location**: `api/grpc/systems/daggerheart/service.go`
- **Issue**: The service file registers all Daggerheart RPC handlers. As the system grows, this becomes a merge conflict hotspot and grows proportionally with the number of operations.
- **Recommendation**: Use a registration pattern where each sub-package provides a `Register(server)` function, and the root service.go calls them in sequence. This distributes the registration.

### Finding 3: System Guard Enforcement Is Correct
- **Severity**: info
- **Location**: `api/grpc/systems/daggerheart/guard/`
- **Issue**: The guard checks that the campaign uses the Daggerheart game system before allowing any Daggerheart-specific operation. This prevents cross-system operations and is enforced at the transport layer.
- **Recommendation**: Clean enforcement point.

### Finding 4: 42 Test Files at Root
- **Severity**: medium
- **Location**: `api/grpc/systems/daggerheart/*_test.go`
- **Issue**: 42 test files at the package root is hard to navigate. Test files should be co-located with the production code they test — if handlers move to sub-packages, their tests should follow.
- **Recommendation**: Move tests to sub-packages alongside their production code.

## Summary Statistics
- Files reviewed: ~65 (60 root + 2 guard + 3 gameplaystores)
- Findings: 4 (0 critical, 1 high, 2 medium, 0 low, 1 info)
