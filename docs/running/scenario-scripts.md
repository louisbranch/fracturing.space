---
title: "Scenario scripts"
parent: "Running"
nav_order: 8
status: canonical
owner: engineering
last_reviewed: "2026-02-26"
---

# Running Lua Scenario Scripts

Lua scenarios can be executed against the game gRPC API for testing, seeding, or playtesting.

## Prerequisites

The game server must be running before running scenarios:

```bash
# Terminal 1: Start devcontainer + watcher-managed local services
make up

# Terminal 2: Run a scenario
go run ./cmd/scenario -scenario internal/test/game/scenarios/basic_flow.lua
```

Using direct Go commands:

```bash
# Terminal 1: Start the game server
go run ./cmd/game

# Terminal 2: Run a scenario
go run ./cmd/scenario -scenario internal/test/game/scenarios/basic_flow.lua
```

Using Compose:

```bash
COMPOSE="docker compose -f docker-compose.yml -f topology/generated/docker-compose.discovery.generated.yml"

# Terminal 1: Start the game service
$COMPOSE up -d game

# Terminal 2: Run a scenario
$COMPOSE --profile tools run --rm scenario -- -scenario internal/test/game/scenarios/basic_flow.lua
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
go run ./cmd/scenario -scenario internal/test/game/scenarios/basic_flow.lua -assert=false
```

In log-only mode, expectation failures are logged but do not stop execution.

## DSL Examples

Create a participant and a character with chaining (defaults: participant role = PLAYER, character kind = PC, control = participant):

```lua
-- Setup
local scene = Scenario.new("demo")
scene:campaign({name = "Demo", system = "DAGGERHEART"})

-- Participant + character
scene:participant({name = "John"}):character({name = "Frodo"})

return scene
```

Use prefab shortcuts for known presets:

```lua
scene:prefab("frodo")
```

## Mock Auth

Scenario runs use a permissive in-process auth helper that generates synthetic user IDs and allows invite-related actions. No auth service is required.
