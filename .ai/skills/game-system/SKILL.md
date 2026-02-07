---
name: game-system
description: Steps and checklists for adding a new game system
user-invocable: true
---

# Adding a New Game System

See `docs/project/game-systems.md` for the comprehensive guide, including design rationale and a VtM example.

## Quick Summary

1. Add enum value to `common/v1/game_system.proto`.
2. Create `internal/systems/{name}/` with domain logic and state handlers.
3. Implement `systems.GameSystem` interface (including `StateFactory` and `OutcomeApplier`).
4. Create protos in `api/proto/systems/{name}/v1/` (mechanics, state, service).
5. Create extension tables in `internal/storage/sqlite/migrations/`.
6. Create gRPC service in `internal/api/grpc/systems/{name}/`.
7. Add MCP domain handlers in `internal/mcp/domain/{name}.go`.
8. Add integration tests.
