# AGENTS.md

Use the guidance below as the default operating rules for agents working here.
If the project structure evolves, update this file to match the real tooling.

## Scope
- Applies to the entire repository.
- If future nested `AGENTS.md` files exist, follow the most specific one.
- Keep instructions concise and action oriented.

## Quick Start
- Initialize a Go module with `go mod init <module>`.
- Keep `go.mod` and `go.sum` in repo root.
- Prefer Go 1.21+ unless specified otherwise.

## Build Commands
- Build all packages: `go build ./...`
- Build a single package: `go build ./path/to/pkg`
- Build a single binary (example): `go build -o bin/app ./cmd/app`
- Verify module tidy: `go mod tidy`

## Test Commands
- Run all tests: `go test ./...`
- Run a single package: `go test ./path/to/pkg`
- Run a single test by name: `go test ./path/to/pkg -run '^TestName$'`
- Run subtests by name: `go test ./path/to/pkg -run 'TestName/Subtest'`
- Run with race detector: `go test -race ./...`
- Run with coverage: `go test -cover ./...`
- Run without cache: `go test -count=1 ./...`

## Lint / Format Commands
- Format all files: `gofmt -w .`
- Format with goimports (recommended): `goimports -w .`
- Vet all packages: `go vet ./...`
- Lint all packages: `golangci-lint run ./...`
- Lint a single package: `golangci-lint run ./path/to/pkg`

## Code Organization
- Put entrypoints in `cmd/web`, `cmd/server`, `cmd/mcp`.
- Keep shared logic in `internal/` (preferred) or `pkg/`.
- Keep package names short and descriptive.
- Keep files focused; split large files by responsibility.
- Treat the MCP layer as a thin transport wrapper; keep rule validation and game logic in the gRPC server/dice packages.

## Imports
- Group imports: standard library, third-party, local.
- Use `goimports` to manage import order and pruning.
- Avoid dot imports.
- Use alias only when necessary for clarity or conflicts.

## Formatting
- Always run `gofmt` (or `goimports`) on edited files.
- Keep line length reasonable; break long expressions.
- Prefer early returns to reduce nesting.

## Naming Conventions
- Use `camelCase` for locals and parameters.
- Use `PascalCase` for exported identifiers.
- Use short, meaningful names; avoid cryptic single-letter names except for idiomatic uses (loop indices, receivers, short-lived locals).
- Name interfaces by behavior (`Reader`, `Store`, `Validator`).
- Name concrete types by domain (`UserStore`, `OAuthClient`).

## Types and Interfaces
- Prefer concrete types in APIs; accept interfaces at boundaries.
- Keep interfaces small and focused.
- Avoid empty interface (`interface{}`); use `any` only when necessary.
- Use type aliases sparingly; prefer defined types for clarity.

## Error Handling
- Return errors explicitly; avoid panics for control flow.
- Wrap errors with `%w` to preserve causes.
- Use sentinel errors for stable comparisons.
- Include context in error messages, no trailing punctuation.
- Prefer `errors.Is` and `errors.As` for checks.

## Logging
- Use structured logging if a logger exists.
- Avoid `fmt.Println` in library code.
- Include useful context fields; avoid dumping large structs.

## Concurrency
- Avoid sharing mutable state without synchronization.
- Prefer context cancellation for goroutines.
- Use `sync.WaitGroup` to coordinate goroutines.
- Avoid goroutine leaks; ensure exit paths are clear.

## Testing Style
- Use table-driven tests for multiple cases.
- Name tests `TestXxx` and subtests with `t.Run`.
- Prefer `t.Helper` for helper functions.
- Keep tests deterministic; avoid real network calls.
- Use fake implementations over heavy mocks.

## Dependency Management
- Avoid adding heavy dependencies without justification.
- Prefer standard library equivalents when possible.
- Keep `go.mod` tidy and committed.

## Documentation
- Document exported types and functions.
- Add module, file, and package comments that explain intent and scope.
- Document core data structures and methods with the "why" (purpose, lifecycle, consumers), not just what they do.
- For security-sensitive flows (auth, crypto, token validation), add short rationale comments explaining the threat being mitigated.
- Keep README short and task focused.
- Document env vars in `README` or `docs/`.

### Documentation Checklist (all code changes)
- Add/update doc comments for any new or modified identifiers (exported or not).
- Ensure package or file comments exist when adding new files or packages.
- Capture behavior or lifecycle changes in existing docs (comments or README) when code changes alter intent.

### Documentation Self-Check (before commit)
- Review touched files for missing doc comments and add why/intent where needed.

## Security
- Validate input at boundaries.
- Avoid `exec.Command` with user input.
- Use `context` with timeouts for external calls.
- When editing auth/crypto flows, add a brief function-level intent comment plus an inline rationale comment at non-obvious checks.

## Cursor / Copilot Rules
- No `.cursor/rules`, `.cursorrules`, or `.github/copilot-instructions.md` found.
- If added later, incorporate their rules here.

## Agent Workflow
- Always create or switch to a new branch before making changes; never work directly on main.
- If you are not on a prefixed work branch (for example: `feat/`, `fix/`, `chore/`, `docs/`), stop and switch before editing files.
- Use branch prefixes: `feat/<name>`, `fix/<name>`, `chore/<name>`, `docs/<name>`.
- Use commit prefixes: `feat:`, `fix:`, `chore:`, `docs:` with a short why-focused subject.
- Match PR titles to the same prefix style (example: `feat: add duality outcome tool`).
- Prefer small, focused changes per request.
- Keep one intent per PR; split unrelated changes.
- Avoid reformatting unrelated code.
- Do not introduce new files unless required.
- Mention any missing tests or tooling in summaries.
- Run `go test ./...` when a task is complete.
- If tests pass, create a commit unless the user says otherwise.
- PR bodies do not need a testing section mentioning `go test ./...`.
- Always use squash when enabling PR auto-merge.

## Versioning
- Follow semantic versioning if releases are introduced.
- Tag releases in git if requested.

## Future Updates
- Update commands once a build system is chosen.
- Add CI references when available.
- Extend sections for database, API, or frontend code.
