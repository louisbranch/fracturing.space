local scn = Scenario.new("environment_mirkwood_blight_charcoal_constructs")
local dh = scn:system("DAGGERHEART")

-- Capture the trampling charcoal constructs hazard.
scn:campaign{
  name = "Environment Mirkwood Blight Charcoal Constructs",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Gandalf")

-- The corrupted animals trample a chosen point.
scn:start_session("Charcoal Constructs")

-- Missing DSL: apply area reaction roll and half damage on success.
dh:reaction_roll{ actor = "Gandalf", trait = "agility", difficulty = 16, outcome = "hope" }

scn:end_session()

return scn
