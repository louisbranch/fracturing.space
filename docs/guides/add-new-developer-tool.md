---
title: "Add a developer tool"
parent: "Guides"
nav_order: 8
---

# Add a developer tool

Use this checklist when introducing a new tool under `internal/tools`.

## 1. Choose the right location

- Use `internal/tools/<name>` for reusable tool logic.
- Use `cmd/<name>` for command wiring if the tool is exposed through a dedicated command.
- Keep runtime helpers in shared tool helper packages only when multiple tools need the same behavior.

## 2. Implement the standard contract

For non-`main` tool packages:

- `type Config struct { ... }`
- `ParseConfig(*flag.FlagSet, []string) (Config, error)`
- `Run(context.Context, Config, stdout, stderr) error`

For package `main` tools:

- `run(args []string, stdout, stderr io.Writer) error`
- `main()` only calls `run` and handles exit.

## 3. Keep boundaries clean

- No `os.Exit` outside `main`.
- No hidden global mutation hooks for tests.
- Put interfaces at consumption points for external dependencies.

## 4. Add meaningful tests

Minimum:

- parse/validation tests
- behavior tests for normal and error paths
- output-format tests for generated artifacts/reports

## 5. Wire quality and docs

- Add or update Make targets when the tool is part of standard workflows.
- Update relevant docs (`docs/running`, `docs/architecture`, `docs/reference`) when behavior is durable.
- Prefer command examples that work in local dev and CI.

## 6. Verify before PR

Run at least:

```bash
go test ./internal/tools/...
make ci-go-tests-local
```
