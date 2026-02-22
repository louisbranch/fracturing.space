# Newcomer Clarity Refactor Execution Plan

Goal: improve newcomer onboarding by making registration wiring and side-effect behavior explicit, testable, and documented.

1. [x] Refactor MCP tool/resource registration into named modules in `internal/services/mcp/service`.
2. [x] Introduce a registration abstraction (`mcpRegistrationTarget`) and adapter to keep tool/resource registration testable without coupling to MCP concrete API.
3. [x] Add focused tests for MCP module composition and isolation (`internal/services/mcp/service/helpers_test.go`).
4. [x] Remove implicit global state from inline projection intent lookup by using an injected resolver abstraction (`internal/services/game/api/grpc/internal/domainwrite/helper.go`).
5. [x] Add contributor-facing documentation describing the MCP module model and domain-write resolver decision path in `docs/`.
6. [x] Add a lightweight contributor checklist in `docs/` for adding a new MCP tool/resource pair (module registration, handler wiring, tests).
7. [x] Add a test that exercises a future registry mismatch path in MCP adapter (nil/unsupported handler panic message path) to avoid silent drift.

## Next-wave continuation (2026-02+)

- [x] Resolve `GetContentCatalog` orchestration maintainability in `internal/services/game/api/grpc/systems/daggerheart/content_service.go`.
- [x] Implement MCP transport hardening seams in `internal/services/mcp/service/http_transport.go` and `internal/services/mcp/service/server.go` (auth, rate-limits, TLS/API-token config seam).
- [x] Continue projection store decomposition under `internal/services/game/storage/sqlite/` without changing schema behavior.
- [x] Split registry bootstrap/validation in `internal/services/game/domain/engine/registries.go` into smaller validator modules.
- [x] Enforce a hard no-migration constraint for this wave (no schema edits, no migration file edits, no migration execution).

## Additional backlog carry-over (monitor-only)

- [x] Evaluate whether to split MCP transport tests after completion of runtime hardening seams (`internal/services/mcp/service/server_test.go`).
- [x] Split remaining large Daggerheart action clusters under `internal/services/game/api/grpc/systems/daggerheart/` when this wave is stable:
  - [x] conditions cluster (`actions_conditions.go` -> `conditionsApplication` orchestration boundary)
  - [x] countdown cluster (`actions_countdowns.go` -> `countdownApplication` orchestration boundary)
  - [x] recovery cluster (`actions_recovery.go` -> `recoveryApplication` orchestration boundary)
  - [x] adversary cluster (`adversaries.go` -> `adversaryApplication` orchestration boundary)
  - [x] damage cluster (`actions_damage.go` -> `damageApplication` orchestration boundary)
- [x] Split Daggerheart session workflow execution cluster out of `actions_session_flows.go` into `sessionFlowApplication` (`session_flow_session_application.go`) and keep action file as explicit delegation wrappers.
- [x] Split Daggerheart outcome workflow execution cluster out of `actions_outcomes.go` into `outcomeApplication` (`outcome_application_session.go`) with delegation wrappers.
  - [x] Move outcome-specific helper surface from `actions_outcomes.go` into `outcome_helpers.go` to keep entrypoint logic delegating-only.
- [x] Add seam behavior tests for MCP `HTTPTransport` request authorization and rate limiting (`internal/services/mcp/service/http_transport_authz_test.go`).
- [x] Add focused tests for MCP TLS seam behavior and `TLSConfig` listener coverage (`internal/services/mcp/service/http_transport_tls_test.go`).
- [x] Split MCP transport auth/seams tests into dedicated `http_transport_authz_test.go` while preserving `server_test.go` for composition wiring.
- [x] Add docs for repository-wide seam-first refactor lessons (when to add composition seams, when to move orchestration into type-owned helpers).
