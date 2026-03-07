local scn = Scenario.new("environment_osgiliath_ruins_buried_knowledge")
local dh = scn:system("DAGGERHEART")

-- Model the haunted city's buried knowledge investigation.
scn:campaign{
  name = "Environment Osgiliath Ruins Buried Knowledge",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A PC investigates the city's lore.
scn:start_session("Buried Knowledge")

-- Info/loot fanout and failure-stress branch remain unresolved.
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 14, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
