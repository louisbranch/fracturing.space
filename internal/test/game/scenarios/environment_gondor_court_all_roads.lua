local scene = Scenario.new("environment_gondor_court_all_roads")

-- Model disadvantage on Presence rolls that resist imperial norms.
scene:campaign{
  name = "Environment Gondor Court All Roads",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Court etiquette hampers dissenting actions.
scene:start_session("All Roads Lead Here")

scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "fear", disadvantage = 1 }
scene:apply_roll_outcome{}

scene:end_session()

return scene
