---
title: "Scenario scripts"
parent: "Running"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-03-10"
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
COMPOSE="docker compose -f docker-compose.yml -f topology/generated/docker-compose.serviceaddr.generated.yml"

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

Campaign defaults:

- `gm_mode` defaults to `HUMAN` when omitted in `scn:campaign({...})`.
- `AI`/`HYBRID` campaign modes require a real campaign AI binding before `start_session`.

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

Scenario suites are part of the public runtime verification surface:

```bash
make test
make smoke
make check
```

Use `make smoke` for quick feedback while iterating and `make check` before
opening or updating a PR. The canonical workflow is documented in
[Verification commands](verification.md).

## Supported verification

For one-off script execution, use the scenario CLI commands above. For supported
project verification, use the public Make surface in
[Verification commands](verification.md).

Optional environment controls:

- `SCENARIO_MANIFEST`: newline-delimited scenario list for selective runs.
- `SCENARIO_ONLY`: comma-separated scenario names/paths.
- `SCENARIO_FILTER`: regex filter for scenario names/paths.

CI may shard scenario coverage internally, but shard-specific targets are not
part of the public contributor command surface.

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
