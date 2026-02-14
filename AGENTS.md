# AGENTS.md

Single source of agent directives and project context.

## Preflight

- At the start of a session, verify you are not on `main` with `git branch --show-current`; if you are on `main`, stop, and ask for instructions.

## ExecPlans

- For complex features or significant refactors, write an ExecPlan and follow `PLANS.md`.
- ExecPlans live in `plans/` and must be kept up to date as work progresses.

## Safety

- Do not commit files containing secrets (.env, credentials).
- Game service writes are event-driven: mutate state by emitting events and applying projections; do not write projection/storage records directly from non-read handlers.

## Documentation culture

- Maintain a domain-driven design vision; keep domain language and boundaries intentional.
- Offer to promote high-level decisions, domain language changes, and architecture evolution to `docs/` .
- Document code, including non exported functions and types, focused on why not how.

## Test-Driven Development (TDD)

- **Invariant**: For behavior changes, follow TDD end-to-end (test first, then minimal implementation, then refactor). Exceptions are limited to non-behavioral changes (docs-only or refactors with no behavior change), which must be explicitly called out.
- **Red**: Write one small test that defines a single behavior and verify it fails before writing any implementation code.
- **Green**: Implement the minimum code necessary to make that test pass—no extra features or generalization.
- **Refactor**: Improve structure and clarity while keeping all tests passing and without changing behavior.
- **Coverage as guardrail**: When adding or changing production code, run `make cover` and report the coverage impact.
- **Behavior tests required**: Add or update tests for new behavior; if a change is test-neutral (docs/refactor), call it out explicitly.
- **Non-regression**: Keep coverage from regressing versus the current baseline; CI enforces non-regression.
- **Generated code**: When introducing new generated outputs, update `COVER_EXCLUDE_REGEX` in `Makefile` so coverage reflects hand-written code.

## Planning sessions

- Create `.ai/plans/<topic>.md` before modifying any other files.
- Plans are session-only and should include a description, task list that is updated as work progresses, plus next steps and out of scope if applicable.
- Before any PR, move lingering knowledge or tasks from `.ai/plans` into `docs/` so it survives the worktree lifecycle.

## Verification

Run `make integration` after code changes (covers full gRPC + MCP + storage path).

```bash
make test        # Unit tests
make integration # Integration tests
make proto       # Regenerate proto code
```

## Commits and PRs

Commit in small, task-sized increments as you work; do not batch everything into a single final commit.

Use matching prefixes with a short, why-focused subject:
- `feat:` - New features
- `fix:` - Bug fixes
- `chore:` - Maintenance
- `docs:` - Documentation

Example: `feat: add duality outcome tool`

## Skills

Load the relevant skill when working in these areas:

Skills live in `.ai/skills/`.

| skill | what | when to use |
| --- | --- | --- |
| `testing` | Test-Driven Development and coverage guardrails | optional deep-dive when guidance is needed |
| `go-style` | Go conventions, build commands, naming, error handling patterns | when editing Go code or running Go build/test |
| `error-handling` | Structured errors and i18n-friendly messaging workflow | when adding or changing error flows/messages |
| `schema` | Database migrations and proto field ordering rules | when editing migrations or protos |
| `game-system` | Steps and checklists for adding a new game system | when implementing a new system |
| `mcp` | MCP tool/resource guidance and parity rules with gRPC | when touching MCP tooling or MCP endpoints |
| `web-server` | Web UI and transport layer conventions | when working on web UI or HTTP transport |
| `pr-issues` | PR review triage, fixes, testing, and auto-merge workflow | when triaging or fixing PR review comments |

## Docs

Use these project docs for architecture and domain guidance:

| doc | what | when to use |
| --- | --- | --- |
| `docs/project/architecture.md` | System architecture, service boundaries, layers | when orienting to overall system design |
| `docs/project/domain-language.md` | Canonical domain terms and naming principles | when naming new APIs/packages/events |
| `docs/project/event-replay.md` | Event journal, replay modes, snapshots | when working on replay, snapshots, maintenance CLI |
| `docs/project/game-systems.md` | Pluggable game-system architecture and add-a-system guide | when adding or modifying a game system |
| `docs/project/participant-invitation.md` | Participant invitation flow and follow-ups | when designing auth or participant flows |
