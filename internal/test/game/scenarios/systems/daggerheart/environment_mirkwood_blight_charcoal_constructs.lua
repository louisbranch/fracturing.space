local scene = Scenario.new("environment_mirkwood_blight_charcoal_constructs")
local dh = scene:system("DAGGERHEART")

-- Capture the trampling charcoal constructs hazard.
scene:campaign{
  name = "Environment Mirkwood Blight Charcoal Constructs",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Gandalf")

-- The corrupted animals trample a chosen point.
scene:start_session("Charcoal Constructs")

-- Missing DSL: apply area reaction roll and half damage on success.
dh:reaction_roll{ actor = "Gandalf", trait = "agility", difficulty = 16, outcome = "hope" }

scene:end_session()

return scene
