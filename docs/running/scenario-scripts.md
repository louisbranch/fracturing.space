---
title: "Scenario scripts"
parent: "Running"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-03-03"
---

# Running Lua Scenario Scripts

Lua scenarios can be executed against the game gRPC API for testing, seeding, or playtesting.

## Prerequisites

The game server must be running before running scenarios:

```bash
# Terminal 1: Start devcontainer + watcher-managed local services
make up

# Terminal 2: Run a scenario
go run ./cmd/scenario -scenario internal/test/game/scenarios/systems/daggerheart/basic_flow.lua
```

Using direct Go commands:

```bash
# Terminal 1: Start the game server
go run ./cmd/game

# Terminal 2: Run a scenario
go run ./cmd/scenario -scenario internal/test/game/scenarios/systems/daggerheart/basic_flow.lua
```

Using Compose:

```bash
COMPOSE="docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml"

# Terminal 1: Start the game service
$COMPOSE up -d game

# Terminal 2: Run a scenario
$COMPOSE --profile tools run --rm scenario -- -scenario internal/test/game/scenarios/systems/daggerheart/basic_flow.lua
```

## CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-grpc-addr` | game server address | `game:8082` |
| `-scenario` | path to scenario lua file | (required) |
| `-assert` | enable assertions (disable to log expectations) | `true` |
| `-verbose` | enable verbose logging | `false` |
| `-timeout` | timeout per step | `10s` |

## Assertion Modes

When assertions are enabled, scenario validations fail fast on mismatches. Disable assertions to turn them into log-only expectations:

```bash
go run ./cmd/scenario -scenario internal/test/game/scenarios/systems/daggerheart/basic_flow.lua -assert=false
```

In log-only mode, expectation failures are logged but do not stop execution.

## DSL Examples

Create a participant and a character with chaining (defaults: participant role = PLAYER, character kind = PC, control = participant):

```lua
-- Setup
local scn = Scenario.new("demo")
local dh = scn:system("DAGGERHEART")
scn:campaign({name = "Demo", system = "DAGGERHEART"})

-- Participant + character
scn:participant({name = "John"}):character({name = "Frodo"})
dh:gm_fear(1)

return scn
```

Use prefab shortcuts for known presets:

```lua
scn:prefab("frodo")
```

Root alias convention:

- canonical semantic name: `scenario`
- preferred shorthand for scripts: `scn`
- avoid `scene` as the root alias to prevent collision with domain `scene` terminology

## Mock Auth

Scenario runs use a permissive in-process auth helper that generates synthetic user IDs and allows invite-related actions. No auth service is required.

## Scenario Test Lanes

Use the scenario-focused Make targets when validating Lua suites:

```bash
make scenario-smoke
make scenario-full
make scenario-fast
```

## Scenario Command Matrix

Use these commands by audience and intent:

| Audience | Command | Use case | When to run |
| --- | --- | --- | --- |
| Users | `make scenario-smoke` | Fast local scenario contract check | During active feature work |
| Users | `make scenario-full` | Full scenario regression sweep | Before merging gameplay/runtime-affecting changes |
| Users | `make scenario-fast` | Faster local full run with parallel workers | When iterating on large scenario suites locally |
| Agents | `make scenario-smoke` | Short feedback loop | After incremental scenario/harness edits |
| Agents | `make scenario-full` | Completion gate for scenario behavior | Before reporting done |
| CI (PR) | `make scenario-smoke` | Fast PR gate for scenario contracts | Every pull request |
| CI (main/nightly) | `SCENARIO_VERIFY_SHARDS_TOTAL=6 make scenario-shard-check` | Ensure shard coverage is complete/non-overlapping | Before shard matrix execution |
| CI (main/nightly) | `SCENARIO_SHARD_TOTAL=6 SCENARIO_SHARD_INDEX=<n> make scenario-shard` | Parallel full scenario fanout | Matrix jobs on non-PR workflows |

Optional environment controls:

- `SCENARIO_MANIFEST`: newline-delimited scenario list for selective runs.
- `SCENARIO_ONLY`: comma-separated scenario names/paths.
- `SCENARIO_FILTER`: regex filter for scenario names/paths.
- `SCENARIO_SHARD_TOTAL` + `SCENARIO_SHARD_INDEX`: deterministic shard selection.

## Scenario Layout

Scenarios are grouped by game system under:

- `internal/test/game/scenarios/systems/<system_id>/*.lua`

Manifest files are stored under:

- `internal/test/game/scenarios/manifests/*.txt`

## Extending To New Systems

Scenarios do not need a new DSL per system. New systems should:

1. add scenarios under `internal/test/game/scenarios/systems/<system_id>/`,
2. register system DSL methods + step dispatch in
   `internal/tools/scenario/system_registry.go`,
3. keep using system handles (`scn:system("<SYSTEM_ID>")`) in Lua scripts.

`scn:campaign` must include `system` explicitly. The runner does not apply an
implicit system default.

Legacy root-level mechanic calls are rejected with migration guidance. Use
system handles (`local sys = scn:system("<SYSTEM_ID>")`) for all
system-owned mechanics.
