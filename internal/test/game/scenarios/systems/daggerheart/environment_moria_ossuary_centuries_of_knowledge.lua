local scn = Scenario.new("environment_moria_ossuary_centuries_of_knowledge")
local dh = scn:system("DAGGERHEART")

-- Capture the knowledge roll in the ossuary library.
scn:campaign{
  name = "Environment Ossuary Centuries of Knowledge",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Research reveals arcana and necromancer plans.
scn:start_session("Centuries of Knowledge")

-- Missing DSL: map outcome to lore details.
dh:action_roll{ actor = "Frodo", trait = "knowledge", difficulty = 19, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
