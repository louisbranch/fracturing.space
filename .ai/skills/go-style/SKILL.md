---
name: go-style
description: Go language conventions including build commands, formatting, naming, error handling, and testing patterns
user-invocable: true
---

# Go Style Guide

Go language conventions for this project.

## Build / Test / Lint

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

- Run `gofmt` or `goimports` on edited files
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

- Return errors explicitly; avoid panics for control flow
- Wrap errors with `%w` to preserve causes
- Use sentinel errors for stable comparisons
- Include context in messages, no trailing punctuation
- Prefer `errors.Is` and `errors.As` for checks

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

- Document exported types and functions
- Add package comments explaining intent and scope
- Document the "why" (purpose, lifecycle), not just what
- For security flows, add rationale comments explaining threats mitigated

## Dependencies

- Avoid heavy dependencies without justification
- Prefer standard library equivalents
- Keep `go.mod` tidy and committed
