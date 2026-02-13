---
name: go-style
description: Go language conventions including build commands, formatting, naming, error handling, and testing patterns
user-invocable: true
---

# Go Style Guide

Go language conventions for this project.

## Build / Test / Lint

Project-wide verification lives in `AGENTS.md`. Use those commands after code changes.

```bash
go build ./...                      # Build all packages
go test ./...                       # Run all tests
go test -race ./...                 # Run with race detector
go test -cover ./...                # Run with coverage
go test ./path/to/pkg -run '^TestName$' # Run single test
go vet ./...                        # Vet all packages
goimports -w .                      # Format with import sorting
go mod tidy                         # Clean up dependencies
```

## Imports

Group imports: standard library, third-party, local. Use `goimports` to manage order. Avoid dot imports. Use aliases only for conflicts.

## Formatting

- Run `goimports` on edited files
- Prefer early returns to reduce nesting
- Keep line length reasonable; break long expressions
- Avoid inline `if err := ...` for multi-line bodies

## Naming

- `camelCase` for locals and parameters
- `PascalCase` for exported identifiers
- Short, meaningful names; avoid cryptic single letters except idiomatically
- Name interfaces by behavior (`Reader`, `Store`, `Validator`)
- Name concrete types by domain (`UserStore`, `OAuthClient`)

## Types and Interfaces

- Prefer concrete types in APIs; accept interfaces at boundaries
- Keep interfaces small and focused
- Use `any` sparingly; prefer defined types

## Error Handling

Use the `error-handling` skill for structured error workflows and messaging rules.

## Logging

- Use structured logging if available
- Avoid `fmt.Println` in library code
- Include useful context fields

## Concurrency

- Avoid sharing mutable state without synchronization
- Prefer context cancellation for goroutines
- Use `sync.WaitGroup` to coordinate goroutines
- Ensure goroutine exit paths are clear

## Testing

- Use table-driven tests for multiple cases
- Name tests `TestXxx` and subtests with `t.Run`
- Use `t.Helper` for helper functions
- Keep tests deterministic; avoid real network calls
- Use fake implementations over heavy mocks

## Documentation

- Document all non-generated exported types and functions
- For unexported helpers, add docs when behavior is complex or domain-critical
- Explain what the function/type does and why it exists; avoid narrating how it works
- Use inline comments to clarify non-obvious steps, invariants, or edge cases
- Add package comments explaining intent and scope
- Create or update `doc.go` when package intent changes; capture responsibilities, boundaries, and non-goals
- For security flows, add rationale comments explaining threats mitigated

## Dependencies

- Avoid heavy dependencies without justification
- Prefer standard library equivalents
- Keep `go.mod` tidy and committed
