local scn = Scenario.new("environment_osgiliath_ruins_apocalypse_then")
local dh = scn:system("DAGGERHEART")

-- Capture the apocalypse replay and its escape countdown.
scn:campaign{
  name = "Environment Osgiliath Ruins Apocalypse Then",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The GM spends Fear to replay a past disaster.
scn:start_session("Apocalypse Then")
dh:gm_fear(1)

-- Example: spend Fear to activate a progress countdown (5).
dh:gm_spend_fear(1):spotlight("Osgiliath Ruins")
dh:scene_countdown_create{ name = "Escape the Apocalypse", kind = "progress", current = 0, max = 5, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 14, outcome = "success_fear" }
dh:apply_roll_outcome{
  on_success = {
    {kind = "scene_countdown_update", name = "Escape the Apocalypse", delta = 1, reason = "escape_progress"},
  },
}

scn:end_session()

return scn
