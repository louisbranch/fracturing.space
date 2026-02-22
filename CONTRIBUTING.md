# Contributing

Thanks for helping improve Fracturing.Space. This guide highlights the
workflow, standards, and expectations used in this repo.

------------------------------------------------------------------------

## Quick Start

1. Fork the repo and create a branch from `main`.
2. Use a prefixed branch name: `feat/`, `fix/`, `chore/`, or `docs/`.
3. Make focused changes with a single intent.
4. Run tests: `go test ./...`.
5. Open a PR with a title using the same prefix style.

------------------------------------------------------------------------

## Build, Test, and Format

- Build all packages: `go build ./...`
- Run all tests: `go test ./...`
- Run integration tests: `go test -tags=integration ./...`
- If `git config --local --get core.hooksPath` is not `.githooks`, run `make setup-hooks` (pre-commit formats staged Go files)
- Verify formatting: `make fmt-check`
- Keep `go.mod` tidy: `go mod tidy`

Integration tests exercise the full gRPC + MCP + storage path. You can also
use the Make targets documented in [integration tests](docs/running/integration-tests.md):

- `make test`
- `make integration`
- `make cover`

------------------------------------------------------------------------

## Code and Structure Guidelines

- Entrypoints belong in `cmd/admin`, `cmd/game`, or `cmd/mcp`.
- Shared logic goes in `internal/` (preferred) or `pkg/`.
- Keep files focused; split large files by responsibility.
- Avoid reformatting unrelated code.
- Prefer early returns to reduce nesting.
- Wrap errors with `%w` and include context in error messages.

------------------------------------------------------------------------

## Where to Put New Features

| Feature Type | Location |
|--------------|----------|
| Campaign settings | `internal/services/game/domain/campaign/` |
| Player/GM management | `internal/services/game/domain/participant/` |
| Character definitions | `internal/services/game/domain/character/` |
| Persistent gameplay state | `internal/services/game/domain/action/`, `internal/services/game/projection/` |
| Session mechanics | `internal/services/game/domain/session/` |
| Game-system-specific rules | `internal/services/game/domain/systems/{system}/` |
| Generic dice mechanics | `internal/services/game/core/dice/` |
| gRPC API endpoints | `internal/services/game/api/grpc/` |
| MCP tools/resources | `internal/services/mcp/domain/` |

------------------------------------------------------------------------

## Adding a New Game System

1. Add enum value to `api/proto/common/v1/game_system.proto`
2. Create `internal/services/game/domain/systems/{name}/` with domain logic
3. Implement `systems.GameSystem` interface
4. Create protos in `api/proto/systems/{name}/v1/`
5. Create gRPC service in `internal/services/game/api/grpc/systems/{name}/`
6. Register MCP tools/resources in `internal/services/mcp/domain/`
7. Add integration tests

See [AGENTS.md](AGENTS.md) for detailed architecture documentation.

------------------------------------------------------------------------

## Documentation Expectations

- Document exported types and functions.
- Add or update doc comments for any modified identifiers.
- Update `docs/` and `README.md` when user-facing behavior changes.
- Keep [docs/index.md](docs/index.md) and README links current.

------------------------------------------------------------------------

## MCP Tool Changes

When adding a new MCP tool, update the expected tools list in
`internal/test/integration/fixtures/blackbox_tools_list.json`.

------------------------------------------------------------------------

## Commits and PRs

- Commit messages use prefixes: `feat:`, `fix:`, `chore:`, `docs:`.
- Keep PRs small and focused; split unrelated changes.
- Do not open PRs from branches used by closed or merged PRs.

## PR Checklist

- No direct state writes without emitting events.
- Added or updated tests for any new mutations.
- `make integration` passes.
