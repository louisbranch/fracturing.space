# Newcomer Clarity Refactor Execution Plan

Goal: improve newcomer onboarding by making registration wiring and side-effect behavior explicit, testable, and documented.

1. [x] Refactor MCP tool/resource registration into named modules in `internal/services/mcp/service`.
2. [x] Introduce a registration abstraction (`mcpRegistrationTarget`) and adapter to keep tool/resource registration testable without coupling to MCP concrete API.
3. [x] Add focused tests for MCP module composition and isolation (`internal/services/mcp/service/helpers_test.go`).
4. [x] Remove implicit global state from inline projection intent lookup by using an injected resolver abstraction (`internal/services/game/api/grpc/internal/domainwrite/helper.go`).
5. [x] Add contributor-facing documentation describing the MCP module model and domain-write resolver decision path in `docs/`.
6. [x] Add a lightweight contributor checklist in `docs/` for adding a new MCP tool/resource pair (module registration, handler wiring, tests).
7. [x] Add a test that exercises a future registry mismatch path in MCP adapter (nil/unsupported handler panic message path) to avoid silent drift.
