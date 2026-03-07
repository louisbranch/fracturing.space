local scene = Scenario.new("environment_isengard_ritual_desecrated_ground")
local dh = scene:system("DAGGERHEART")

-- Model the Hope die reduction from desecrated ground.
scene:campaign{
  name = "Environment Isengard Ritual Desecrated Ground",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The environment suppresses Hope while the ritual site remains tainted.
scene:start_session("Desecrated Ground")

-- Example: reduce Hope Die to d10 until a progress countdown clears it.
dh:countdown_create{ name = "Cleanse Desecration", kind = "progress", current = 0, max = 6, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 14, outcome = "success_hope" }
dh:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "Cleanse Desecration", delta = 1, reason = "cleansing_progress"},
  },
}

scene:end_session()

return scene
