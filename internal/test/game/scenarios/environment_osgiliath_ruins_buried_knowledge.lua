local scene = Scenario.new("environment_osgiliath_ruins_buried_knowledge")

-- Model the haunted city's buried knowledge investigation.
scene:campaign{
  name = "Environment Osgiliath Ruins Buried Knowledge",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A PC investigates the city's lore.
scene:start_session("Buried Knowledge")

-- Info/loot fanout and failure-stress branch remain unresolved.
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 14, outcome = "hope" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
