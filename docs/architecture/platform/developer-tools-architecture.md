---
title: "Developer tools architecture"
parent: "Platform surfaces"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-03-07"
---

# Developer tools architecture

Canonical architecture for contributor-facing tooling under `internal/tools`.

## Intent

- Keep operational tooling explicit, testable, and maintainable.
- Make each tool easy to discover, run, and safely evolve.
- Avoid monolithic utility binaries with mixed concerns.

## Boundary model

`internal/tools` is split by responsibility:

- **Runtime helpers** (`internal/tools/cli`): shared signal and CLI helper seams.
- **Small one-shot tools**: docs/check/report style commands (for example `i18ncheck`, `topologygen`, `eventdocgen`).
- **Workflow tools**: higher-complexity orchestration (`seed`, `scenario`, `maintenance`, `importer`).

Tool packages should own business logic and remain transport-neutral.
Command entrypoints (`cmd/*` or package `main`) should only parse flags, wire dependencies, and exit.

Workflow tools should also expose explicit internal seams:

- Keep parser/DSL loading isolated from runtime side effects (for example `internal/tools/scenario/script`).
- Keep system adapters and registry declarations in dedicated files/packages, separate from registry indexing/validation.
- Isolate persistence side effects behind injected interfaces (for example seed state loading/saving) so idempotency behavior is testable without filesystem coupling.

## Command contract

Every tool should converge on the same shape:

1. `Config` type for user-facing inputs.
2. `ParseConfig(*flag.FlagSet, []string) (Config, error)` where the package is not `main`.
3. `Run(context.Context, Config, stdout, stderr) error` (or `run(args, stdout, stderr) error` for package `main` tools).
4. `main()` handles `os.Exit` only after `Run`/`run` returns.

Rules:

- Return errors from logic; avoid `os.Exit` in non-`main` functions.
- Keep stdout machine-consumable where possible; route diagnostics to stderr.
- Define interfaces at consumption seams for external IO/process/network dependencies.

## Testability rules

- Unit-test parsing, normalization, and report rendering without process exits.
- Keep integration tests focused on tool seam contracts (filesystem, process, gRPC, SQLite).
- Use dependency injection instead of mutable package globals for test doubles.

## Ownership and docs

Durable tool behavior and architectural decisions must live in `docs/`.
Temporary execution notes stay in `.agents/plans/` and should be promoted or deleted when complete.

See also: [Add a developer tool](../../guides/add-new-developer-tool.md).
