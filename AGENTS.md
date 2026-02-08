# AGENTS.md

Single source of agent directives and project context.

## Safety

- Never work directly on main; create a feature branch first.
- Run tests (`make integration`) before committing.
- Do not commit files containing secrets (.env, credentials).
- Do not push to closed/merged PR branches; open new ones.
- Prefer squash merge when enabling auto-merge.

## Learning Workflow

Capture and crystallize learnings to improve future sessions.

### Diary Entries (`/diary`)

At the end of meaningful sessions, use `/diary` to capture:
- Design decisions and rationale
- Challenges and solutions
- Patterns discovered
- Future considerations

Entries stored in `.ai/memory/diary/`. Skip for trivial sessions.

### Reflection (`/reflect`)

Periodically use `/reflect` to:
- Analyze accumulated diary entries
- Identify recurring patterns
- Propose AGENTS.md updates

## Project Overview

Fracturing.Space: server-authoritative mechanics and campaign-state service for tabletop RPG campaigns.
Primary use case is enabling an AI Game Master.

Supports multiple game systems (Daggerheart first, with architecture for D&D 5e, VtM, etc.).

## Architecture

### Three-Layer Design

- **Transport**: gRPC server (`cmd/server`) + MCP bridge (`cmd/mcp`) + Web UI (`cmd/web`)
- **Domain**: Game systems (`internal/systems/`) + Campaign model (`internal/campaign/`)
- **Storage**: SQLite persistence (`data/fracturing.space.db`)

MCP is a thin transport wrapper; all rules and state logic live in gRPC/domain packages.

### Campaign Model

Campaign data is organized into three tiers by change frequency:

| Layer | Subpackages | Changes | Contents |
|-------|-------------|---------|----------|
| **Campaign** (Config) | `campaign/`, `campaign/participant/`, `campaign/character/` | Setup time | Name, system, GM mode, participants, character profiles |
| **Snapshot** | `campaign/snapshot/` | At any event sequence | Materialized projection cache for replay/performance |
| **Session** (Gameplay) | `campaign/session/` | Every action | Active session, events, rolls, outcomes |

### Game System Architecture

- Each game system is a plugin under `internal/systems/`.
- Game system gRPC services live in `internal/api/grpc/systems/{name}/`.
- Systems are registered at startup and campaigns are bound to one system at creation.

### Key Packages

| Package | Responsibility |
|---------|----------------|
| `internal/core/dice/` | Generic dice rolling primitives |
| `internal/core/check/` | Difficulty check primitives |
| `internal/core/random/` | Cryptographic seed generation |
| `internal/systems/daggerheart/` | Daggerheart/Duality dice mechanics |
| `internal/campaign/` | Campaign configuration and lifecycle |
| `internal/campaign/participant/` | Player and GM management |
| `internal/campaign/character/` | Character profiles and controllers |
| `internal/campaign/snapshot/` | Snapshot projections (char state, GM fear) |
| `internal/campaign/session/` | Session lifecycle and events |
| `internal/api/grpc/` | gRPC service implementations |
| `internal/mcp/` | MCP tool/resource handlers |
| `internal/storage/` | Persistence interfaces |
| `internal/telemetry/` | Events and metrics (placeholder) |

### Proto Structure

```
api/proto/
├── common/v1/               # Shared types (RNG, GameSystem enum)
├── campaign/v1/             # System-agnostic campaign model
│   ├── campaign.proto       # Campaign + CampaignService
│   ├── session.proto        # Session + SessionService
│   ├── snapshot.proto       # Snapshot + SnapshotService
│   ├── participant.proto
│   └── character.proto
└── systems/daggerheart/v1/  # Daggerheart mechanics
    ├── mechanics.proto      # Duality dice, outcomes
    └── service.proto        # DaggerheartService
```

## Verification

Run `make integration` after changes (covers full gRPC + MCP + storage path).

```bash
make test        # Unit tests
make integration # Integration tests
make proto       # Regenerate proto code
```

## Skills

Load the relevant skill when working in these areas:

Skills live in `.ai/skills/` (with a symlink at `.claude/skills/` for tool compatibility).

- `workflow`: Git branching, commits, and PR conventions.
- `go-style`: Go conventions, build commands, naming, error handling patterns.
- `error-handling`: Structured errors and i18n-friendly messaging workflow.
- `schema`: Database migrations and proto field ordering rules.
- `game-system`: Steps and checklists for adding a new game system.
- `mcp`: MCP tool/resource guidance and parity rules with gRPC.
- `web-server`: Web UI and transport layer conventions.
- `diary`: Capture session learnings.
- `reflect`: Analyze diaries and update agent guidance.
