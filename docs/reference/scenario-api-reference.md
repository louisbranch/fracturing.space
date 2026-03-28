---
title: "Scenario API reference"
parent: "Reference"
nav_order: 25
status: canonical
owner: engineering
last_reviewed: "2026-03-27"
---

# Scenario API reference

Scenarios are Lua scripts that exercise end-to-end gameplay through the real
gRPC service stack. They live under `internal/test/game/scenarios/` and run as
part of `make smoke`.

## Running scenarios

```bash
# All scenarios (integration + scenario lanes)
make smoke

# Single scenario file (via go test, matching the runner)
go test ./internal/test/game/... -run TestScenario/basic_flow -count=1
```

## Core DSL

Every scenario starts with a constructor and returns the scenario object:

```lua
local scn = Scenario.new("my_scenario")
local dh = scn:system("DAGGERHEART")
-- ...steps...
return scn
```

### Setup phase

| Function | Purpose |
|----------|---------|
| `Scenario.new(name)` | Create a named scenario. |
| `scn:system(id)` | Obtain a system handle for system-specific helpers (e.g. `"DAGGERHEART"`). |
| `scn:campaign{name, system, gm_mode, theme}` | Create the campaign. |
| `scn:participant(name, opts)` | Join a participant (GM or player). |
| `scn:pc(name, opts)` | Create a player character (auto-joins a participant if needed). |
| `scn:npc(name, opts)` | Create a non-player character. |
| `scn:prefab(name, opts)` | Create a character from a prefab template. |

### Session lifecycle

| Function | Purpose |
|----------|---------|
| `scn:start_session(name)` | Start a session (auto-creates if needed). |
| `scn:end_session()` | End the current session. |

### Scene management

| Function | Purpose |
|----------|---------|
| `scn:create_scene{name, ...}` | Open a new scene. |
| `scn:end_scene()` | Close the current scene. |
| `scn:scene_add_character(name)` | Add a character to the active scene. |
| `scn:scene_remove_character(name)` | Remove a character from the active scene. |
| `scn:scene_transfer_character(name, opts)` | Transfer a character between scenes. |
| `scn:scene_transition{...}` | Transition the scene (change phase, setting, etc.). |
| `scn:set_spotlight(target)` | Set the campaign spotlight. |
| `scn:clear_spotlight()` | Clear the campaign spotlight. |

### Interaction (player/GM turns)

| Function | Purpose |
|----------|---------|
| `scn:interaction_activate_scene(...)` | Activate a scene for interaction. |
| `scn:interaction_open_scene_player_phase(...)` | Open a player phase. |
| `scn:interaction_submit_scene_player_action(...)` | Submit a player action. |
| `scn:interaction_yield_scene_player_phase(...)` | Player yields their phase. |
| `scn:interaction_interrupt_scene_player_phase(...)` | GM interrupts the player phase. |
| `scn:interaction_resolve_scene_player_review(...)` | Resolve a review step. |
| `scn:interaction_record_scene_gm_interaction(...)` | Record a GM interaction. |
| `scn:interaction_open_session_ooc(...)` | Pause for out-of-character discussion. |
| `scn:interaction_post_session_ooc(...)` | Post an OOC message. |
| `scn:interaction_resolve_session_ooc(...)` | Resolve the OOC pause. |
| `scn:interaction_expect{...}` | Assert interaction state. |

### System-specific (Daggerheart)

Obtained via `dh = scn:system("DAGGERHEART")`:

| Function | Purpose |
|----------|---------|
| `dh:adversary(name, opts)` | Create an adversary. |
| `dh:attack{actor, target, trait, ...}` | Execute an attack roll with outcome assertions. |
| `dh:expect_gm_fear{...}` | Assert GM fear value. |
| `Modifiers.mod(source, value)` | Build a roll modifier. |
| `Modifiers.hope(source, amount)` | Build a hope spend descriptor. |

## Example

A minimal scenario that creates a campaign, adds a PC, and runs a quiet
session:

```lua
local scn = Scenario.new("basic_flow")
scn:system("DAGGERHEART")

scn:campaign{
  name   = "Basic Flow Campaign",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme  = "basics"
}

scn:pc("Frodo")
scn:start_session("First Session")
scn:end_session()

return scn
```

A combat scenario with outcome assertions:

```lua
local scn = Scenario.new("action_roll_outcomes")
local dh = scn:system("DAGGERHEART")

scn:campaign{ name = "Outcomes", system = "DAGGERHEART", gm_mode = "HUMAN" }
scn:pc("Frodo", { stress = 1 })
dh:adversary("Nazgul")

scn:start_session("Combat")

dh:attack{
  actor = "Frodo", target = "Nazgul", trait = "instinct",
  difficulty = 0, outcome = "hope",
  expect_outcome = "hope", expect_hope_delta = 1,
  expect_damage_total = 5, damage_type = "physical"
}

scn:end_session()
return scn
```
