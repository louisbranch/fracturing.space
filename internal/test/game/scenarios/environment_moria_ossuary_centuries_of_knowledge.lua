local scene = Scenario.new("environment_moria_ossuary_centuries_of_knowledge")

-- Capture the knowledge roll in the ossuary library.
scene:campaign{
  name = "Environment Ossuary Centuries of Knowledge",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Research reveals arcana and necromancer plans.
scene:start_session("Centuries of Knowledge")

-- Missing DSL: map outcome to lore details.
scene:action_roll{ actor = "Frodo", trait = "knowledge", difficulty = 19, outcome = "hope" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
