local scn = Scenario.new("environment_gondor_court_all_roads")
local dh = scn:system("DAGGERHEART")

-- Model disadvantage on Presence rolls that resist imperial norms.
scn:campaign{
  name = "Environment Gondor Court All Roads",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Court etiquette hampers dissenting actions.
scn:start_session("All Roads Lead Here")

dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "fear", disadvantage = 1 }
dh:apply_roll_outcome{}

scn:end_session()

return scn
