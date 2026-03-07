local scn = Scenario.new("environment_mirkwood_blight_indigo_flame")
local dh = scn:system("DAGGERHEART")

-- Capture the knowledge roll about the indigo flame corruption.
scn:campaign{
  name = "Environment Mirkwood Blight Indigo Flame",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Gandalf")

-- The party studies the corrupted tree.
scn:start_session("Indigo Flame")

-- Story-detail fanout and optional stress-for-extra-clue remain unresolved.
dh:action_roll{ actor = "Gandalf", trait = "knowledge", difficulty = 16, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
