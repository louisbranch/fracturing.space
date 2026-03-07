local scn = Scenario.new("environment_old_forest_grove_overgrown")
local dh = scn:system("DAGGERHEART")

-- Capture the Overgrown Battlefield investigation in the Old Forest Grove.
scn:campaign{
  name = "Environment Old Forest Grove Overgrown",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The grove reveals its history on a successful Instinct roll.
scn:start_session("Old Forest Grove")

-- Example: success with Hope learns all details; failure can mark Stress for one.
-- Graded information fanout and stress option remain unresolved.
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 11, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
