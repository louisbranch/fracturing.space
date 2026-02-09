# AGENTS.md

Single source of agent directives and project context.

## Preflight

- Before any changes (including plans), verify you are not on `main` with `git branch --show-current`; if you are on `main`, create or switch to a feature branch before proceeding (see the `workflow` skill for branch conventions).

## Safety

- Do not commit files containing secrets (.env, credentials).

## Documentation culture

- Maintain a domain-driven design vision in the docs; keep domain language and boundaries intentional.
- Use `docs/` for high-level decisions, domain language changes, and architecture evolution.
- Keep code comments focused on the why for non-obvious behavior; promote durable knowledge to `docs/`.

## Planning sessions

- Then create `.ai/plans/<topic>.md` before modifying any other files.
- Plans are session-only and should include a description, task list that is updated as work progresses, plus next steps and out of scope if applicable.
- Before any PR, move lingering knowledge or tasks from `.ai/plans` into `docs/` so it survives the worktree lifecycle.

## Verification

Run `make integration` after code changes (covers full gRPC + MCP + storage path).

```bash
make test        # Unit tests
make integration # Integration tests
make proto       # Regenerate proto code
```

## Skills

Load the relevant skill when working in these areas:

Skills live in `.ai/skills/`.

| skill | what | when to use |
| --- | --- | --- |
| `workflow` | Git branching, commits, and PR conventions | when doing git workflows or preparing commits/PRs |
| `go-style` | Go conventions, build commands, naming, error handling patterns | when editing Go code or running Go build/test |
| `error-handling` | Structured errors and i18n-friendly messaging workflow | when adding or changing error flows/messages |
| `schema` | Database migrations and proto field ordering rules | when editing migrations or protos |
| `game-system` | Steps and checklists for adding a new game system | when implementing a new system |
| `mcp` | MCP tool/resource guidance and parity rules with gRPC | when touching MCP tooling or MCP endpoints |
| `web-server` | Web UI and transport layer conventions | when working on web UI or HTTP transport |

## Docs

Use these project docs for architecture and domain guidance:

| doc | what | when to use |
| --- | --- | --- |
| `docs/project/architecture.md` | System architecture, service boundaries, layers | when orienting to overall system design |
| `docs/project/domain-language.md` | Canonical domain terms and naming principles | when naming new APIs/packages/events |
| `docs/project/event-replay.md` | Event journal, replay modes, snapshots | when working on replay, snapshots, maintenance CLI |
| `docs/project/game-systems.md` | Pluggable game-system architecture and add-a-system guide | when adding or modifying a game system |
| `docs/project/auth-participant-vision.md` | Auth/participant model vision, flows, and phases | when designing auth or participant flows |
