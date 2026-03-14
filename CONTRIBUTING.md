# Contributing

Thanks for helping improve Fracturing.Space. This guide highlights the
workflow, standards, and expectations used in this repo.

------------------------------------------------------------------------

## Quick Start

1. Fork the repo and create a branch from `main`.
2. Use a prefixed branch name: `feat/`, `fix/`, `chore/`, or `docs/`.
3. Make focused changes with a single intent.
4. Read the canonical architecture docs for the area you are changing before editing.
5. Run verification for the slice you changed, then broaden to the repo-level targets before opening or updating a PR.
6. Open a PR with a title using the same prefix style.

------------------------------------------------------------------------

## Build, Test, and Format

- Build all packages: `go build ./...`
- Run all tests: `make test`
- If `git config --local --get core.hooksPath` is not `.githooks`, run `make setup-hooks` (pre-commit formats staged Go files)
- Verify formatting: `make fmt-check`
- Keep `go.mod` tidy: `go mod tidy`

Use the supported verification commands documented in
[verification commands](docs/running/verification.md):

- `make test`
- `make smoke`
- `make check`
- `make cover`
- `make web-architecture-check` (required when changing `internal/services/web/` architecture, modules, routes, or templates)
- `make game-architecture-check` (required when changing `internal/services/game/` domain boundaries or write-path architecture guards)

For web changes, also read:

- [Web architecture](docs/architecture/platform/web-architecture.md)
- [Web contributor map](docs/architecture/platform/web-contributor-map.md)
- [Web module playbook](docs/guides/web-module-playbook.md)

------------------------------------------------------------------------

## Code and Structure Guidelines

- Entrypoints belong in `cmd/*` and should stay thin (`internal/cmd/*` owns
  flag/env parsing and runtime orchestration).
- Shared logic goes in `internal/` (preferred) or `pkg/`.
- Keep files focused; split large files by responsibility.
- Prefer architecture-first refactors over small edits that worsen boundaries.
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
| Game-system-specific rules | `internal/services/game/domain/bridge/{system}/` |
| Generic dice mechanics | `internal/services/game/core/dice/` |
| gRPC API endpoints | `internal/services/game/api/grpc/` |
| MCP tools/resources | `internal/services/mcp/domain/` |

------------------------------------------------------------------------

## Adding a New Game System

1. Add enum value to `api/proto/common/v1/game_system.proto`
2. Create `internal/services/game/domain/bridge/{name}/` with domain logic
3. Implement `bridge.GameSystem` interface
4. Create protos in `api/proto/systems/{name}/v1/`
5. Create gRPC service in `internal/services/game/api/grpc/systems/{name}/`
6. Register MCP tools/resources in `internal/services/mcp/domain/`
7. Add integration tests

Use the canonical docs paths: [System extension onboarding](docs/guides/adding-command-event-system.md), [Architecture index](docs/architecture/index.md), and [Domain language](docs/architecture/foundations/domain-language.md).

------------------------------------------------------------------------

## Documentation Expectations

- Document exported types and functions.
- Add or update doc comments for any modified identifiers.
- Update `docs/` and `README.md` when user-facing behavior changes.
- Keep [docs/index.md](docs/index.md) and README links current.
- Promote durable web boundary decisions to the architecture docs instead of leaving them only in code or temporary notes.

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
- `make check` passes.
