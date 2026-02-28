# AGENTS.md

Agent directives for architecture-first, maintainable engineering.

## Priority Order

When trade-offs are required, prefer this order:

1. Clear architecture and domain boundaries.
2. Long-term maintainability and testability.
3. Correctness and operational safety.
4. Delivery speed.
5. Internal backward compatibility.

External/API compatibility is a product decision, not a default technical constraint.

## Preflight

- For non-trivial work, read relevant canonical docs in `docs/architecture/` (and `docs/running/` when runtime behavior is involved) before editing.

## Engineering Posture

- Optimize for codebase health over "just completing the next ticket".
- If the requested micro-change would worsen architecture, propose and prefer the architectural path.
- Favor deletion over accumulation: remove obsolete code, tests, and compatibility layers quickly.

## Architecture-First Refactor Strategy

Use this when existing structure fights the target design:

1. Define target boundaries and package responsibilities.
2. Build the new package/feature path in parallel (clean structure first).
3. Port behavior behind stable contracts (tests at package seams or integration level).
4. Switch callers to the new path.
5. Delete old code paths and stale tests.

Rules:

- Internal compatibility shims are temporary and must include removal criteria.
- Do not keep legacy abstractions "just in case".
- Prefer one clean cutover over indefinite dual-path maintenance.

## Go Heuristics

- Keep packages cohesive and acyclic; design around domain boundaries.
- Define interfaces at consumption points; avoid speculative interfaces.
- Prefer explicit constructors and enforce invariants early.
- Pass `context.Context` first for request-scoped work.
- Return rich errors with context; reserve sentinel errors for real branching needs.
- Keep functions small and intention-revealing; optimize readability before cleverness.
- Inject time/IO/randomness dependencies for deterministic tests.

## Testing Policy (Meaningful Over Ritual)

- Tests should protect durable behavior, invariants, and contracts.
- Use the right level of tests:
  - unit tests for deterministic domain logic,
  - integration tests for component seams and workflows,
  - end-to-end coverage only for critical user/system paths.
- Prefer test-first when it improves design or confidence; avoid ceremonial red/green scripts.
- If behavior is intentionally removed, remove stale tests instead of preserving historical expectations.
- Avoid brittle tests that lock internal implementation details.
- Coverage is a guardrail, not a target to game.

Verification expectations after code changes:

```bash
make test
make integration
```

Command guidance for agents:

- Fast feedback during implementation:
  - `make integration-smoke`
  - `make scenario-smoke`
- Full verification before declaring runtime test changes done:
  - `make integration`
  - `make scenario-full` (or shard-equivalent when validating CI fanout behavior)
- CI parity checks for integration sharding:
  - `INTEGRATION_VERIFY_SHARDS_TOTAL=4 make integration-shard-check`
  - `INTEGRATION_SHARD_TOTAL=4 INTEGRATION_SHARD_INDEX=<n> make integration-shard`

Run `make cover` when production behavior changes and report notable coverage impact.

## Documentation and Knowledge Durability

- Document both exported and non-exported functions/types with "why" context.
- Promote durable decisions (architecture, domain language, migration rationale) to `docs/`.
- Treat `.agents/plans/` notes as temporary working memory; migrate lasting knowledge before PR.
- Keep domain language intentional and consistent with `docs/architecture/domain-language.md`.
- Docs lifecycle policy:
  - Keep `docs/` reader-first and durable; remove stale or completed implementation notes instead of preserving them as historical clutter.
  - Keep in-progress plans, phase trackers, and working backlogs in `.agents/plans/`, not in reader-facing docs.
  - Use section intent consistently: `docs/architecture/` for canonical design, `docs/running/` for runtime operations, `docs/reference/` and `docs/events/` for contracts/generated reference, and `docs/specs/` for executable smoke/QA specs.

## Planning and Execution

- For complex features or significant refactors, create an ExecPlan in `.agents/plans/<topic>.md` before editing code.
- Keep plan task lists current as work progresses.
- Include explicit out-of-scope notes to prevent accidental scope creep.

## Skills

Load the relevant skill when work enters one of these areas:

| skill | focus | when to use |
| --- | --- | --- |
| `testing` | Meaningful testing strategy and coverage guardrails | when deciding test scope, assertions, or coverage trade-offs |
| `architecture-refactor` | Parallel-path refactor and cutover workflow | when incremental edits worsen boundaries or compatibility glue starts spreading |
| `go-style` | Go conventions, naming, package boundaries, and docs | when editing Go code or restructuring packages |
| `error-handling` | Structured errors and i18n-friendly messaging | when adding/changing domain or transport error paths |
| `schema` | Migration/proto change policy and compatibility decisions | when editing migrations, SQL schema, or proto contracts |
| `game-system` | New game-system implementation workflow | when adding/changing game systems or manifest registration |
| `mcp` | MCP transport boundaries and gRPC parity rules | when touching MCP tools/resources/handlers |
| `web-server` | Web transport and feature-boundary conventions | when changing HTTP handlers, routes, or rendering flow |
| `pr-issues` | PR review triage and merge workflow | when triaging/fixing review comments on an existing PR |
| `playwright-cli` | Browser automation commands and workflows | when interacting with web UIs, screenshots, forms, or extraction |

## Project Safety Constraints

- Never commit secrets (`.env`, credentials, tokens).
- Game service writes are event-driven: emit domain events and project state; do not write projection/storage records directly from non-read handlers.
- Prefer safe, reversible operations; avoid destructive git actions unless explicitly requested.

## Completion Criteria

A change is done when:

- architecture is cleaner than before,
- tests validate meaningful behavior at the correct seams,
- obsolete paths are removed,
- documentation is updated where knowledge should persist.

## Commit Guidance

- Commit in small, coherent increments.
- Use concise, why-focused subjects:
  - `feat:` new capability
  - `fix:` behavior correction
  - `chore:` maintenance/refactor/tooling
  - `docs:` documentation-only changes
