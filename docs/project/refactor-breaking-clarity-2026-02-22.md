# Breaking Refactor Outcomes (2026-02-22)

This note records the breaking refactor outcomes executed from the master plan in `.agents/plans/refactor-master-breaking-clarity.md`.

## Scope Executed

1. Daggerheart content gRPC endpoints moved to full descriptor composition for shared validation, filtering, pagination, localization, and error mapping.
2. Game write-path handlers standardized on `internal/services/game/api/grpc/internal/commandbuild` for command envelope construction.
3. Participant and Daggerheart deciders moved to explicit dispatch-table routing with parity tests.
4. AI/Auth storage monoliths split into focused modules for high-churn domains.
5. Admin dashboard activity fan-out/sort moved from handler to a dedicated read composition service.
6. MCP registration switched from a monolithic type-switch to descriptor-table registrars.

## Verification

1. `make test` passed.
2. `make integration` passed.
3. `make cover` passed with total coverage `78.4%`.

## Follow-ups

1. Continue AI/Auth store extraction for remaining domains beyond credentials/passkeys.
2. Evaluate descriptor-based orchestration for `GetContentCatalog` if future maintenance pressure justifies it.
3. Keep MCP non-blocking bootstrap semantics for `newGRPCConn`; revisit only with explicit startup-behavior requirements.
