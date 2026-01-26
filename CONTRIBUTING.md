# Contributing

Thanks for helping improve Duality Engine. This guide highlights the
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
- Format code: `goimports -w .`
- Keep `go.mod` tidy: `go mod tidy`

Integration tests exercise the full gRPC + MCP + storage path. You can also
use the Make targets documented in `docs/integration-tests.md`:

- `make test`
- `make integration`
- `make cover`

------------------------------------------------------------------------

## Code and Structure Guidelines

- Entrypoints belong in `cmd/web`, `cmd/server`, or `cmd/mcp`.
- Shared logic goes in `internal/` (preferred) or `pkg/`.
- Keep files focused; split large files by responsibility.
- Avoid reformatting unrelated code.
- Prefer early returns to reduce nesting.
- Wrap errors with `%w` and include context in error messages.

------------------------------------------------------------------------

## Documentation Expectations

- Document exported types and functions.
- Add or update doc comments for any modified identifiers.
- Update `docs/` and `README.md` when user-facing behavior changes.
- Keep `docs/index.md` and README links current.

------------------------------------------------------------------------

## MCP Tool Changes

When adding a new MCP tool, update the expected tools list in
`internal/integration/mcp_tools_test.go`.

------------------------------------------------------------------------

## Commits and PRs

- Commit messages use prefixes: `feat:`, `fix:`, `chore:`, `docs:`.
- Keep PRs small and focused; split unrelated changes.
- Do not open PRs from branches used by closed or merged PRs.
