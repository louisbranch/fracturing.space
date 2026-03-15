---
title: "Scenario scripts"
parent: "Running"
nav_order: 9
status: canonical
owner: engineering
last_reviewed: "2026-03-12"
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

Use `as = "<participant alias>"` on any core or system step when the scenario
needs to execute that write as a specific participant instead of the campaign
owner. This is how interaction loops model alternating GM/player authority:

```lua
-- GM opens the beat.
scn:interaction_start_player_phase({
  scene = "The Bridge",
  frame_text = "The bridge lurches in the wind. What do you do?",
  characters = {"Aria", "Corin"},
  as = "Guide",
})

-- One player commits a summary and then takes a real system action.
scn:interaction_post({
  as = "Rhea",
  summary = "Aria grabs the near rope before the bridge twists away.",
  characters = {"Aria"},
})
dh:action_roll({
  as = "Rhea",
  actor = "Aria",
  trait = "agility",
  difficulty = 12,
  outcome = "success_fear",
})
```

Interaction scenarios now execute directly through `game.v1.InteractionService`.
Available root interaction steps are:

- `interaction_set_gm_authority`
- `interaction_set_active_scene`
- `interaction_start_player_phase`
- `interaction_post`
- `interaction_yield`
- `interaction_unyield`
- `interaction_accept_player_phase`
- `interaction_request_revisions`
- `interaction_end_player_phase`
- `interaction_pause_ooc`
- `interaction_post_ooc`
- `interaction_ready_ooc`
- `interaction_clear_ready_ooc`
- `interaction_resume_ooc`
- `interaction_expect`

`interaction_expect` reads authoritative interaction state and can assert the
active session/scene, phase status/frame, acting characters or participants,
player slots, OOC state, OOC posts, ready-to-resume set, and GM authority.

Player slot assertions replace the older `posts` and `yielded_participants`
shape. Each slot entry may assert:

- `participant`
- `summary` or `summary_text`
- `characters`
- `yielded`
- `review_status`
- `review_reason`
- `review_characters`

Scene phase status assertions now also support `GM_REVIEW` in addition to the
GM-owned and player-owned phase states.

Example review-return flow:

```lua
scn:interaction_expect({
  scene = "Flooded Archive",
  phase_status = "GM_REVIEW",
  slots = {
    {
      participant = "Rhea",
      summary = "Aria braces the fallen shelf against the door.",
      characters = {"Aria"},
      yielded = true,
      review_status = "UNDER_REVIEW",
    },
  },
})

scn:interaction_request_revisions({
  as = "Guide",
  scene = "Flooded Archive",
  revisions = {
    {
      participant = "Rhea",
      reason = "Keep the lantern dry and tell me where Aria ends up.",
      characters = {"Aria"},
    },
  },
})

scn:interaction_expect({
  scene = "Flooded Archive",
  phase_status = "PLAYERS",
  slots = {
    {
      participant = "Rhea",
      summary = "Aria braces the fallen shelf against the door.",
      characters = {"Aria"},
      review_status = "CHANGES_REQUESTED",
      review_reason = "Keep the lantern dry and tell me where Aria ends up.",
      review_characters = {"Aria"},
    },
  },
})
```

Any scenario step may also assert an expected failure without aborting the
script by adding:

```lua
expect_error = {
  code = "FAILED_PRECONDITION",
  contains = "scene is not the active scene",
}
```

`code` is required and matched against the returned gRPC status code.
`contains` is optional and matched as a substring of the gRPC status message.

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

## Acceptance-first interaction scenarios

The interaction corpus under `internal/test/game/scenarios/systems/daggerheart`
now executes directly through `game.v1.InteractionService`, including invalid
flow contracts that assert expected gRPC failures.

Forward-looking acceptance-only scenario files are still allowed for future
slices, but they should be treated as an exception rather than the default once
runner support exists for a contract.
