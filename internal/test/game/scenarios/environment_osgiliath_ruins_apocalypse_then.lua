local scene = Scenario.new("environment_osgiliath_ruins_apocalypse_then")

-- Capture the apocalypse replay and its escape countdown.
scene:campaign{
  name = "Environment Osgiliath Ruins Apocalypse Then",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The GM spends Fear to replay a past disaster.
scene:start_session("Apocalypse Then")
scene:gm_fear(1)

-- Example: spend Fear to activate a progress countdown (5).
scene:gm_spend_fear(1):spotlight("Osgiliath Ruins")
scene:countdown_create{ name = "Escape the Apocalypse", kind = "progress", current = 0, max = 5, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "agility", difficulty = 14, outcome = "success_fear" }
scene:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "Escape the Apocalypse", delta = 1, reason = "escape_progress"},
  },
}

scene:end_session()

return scene
