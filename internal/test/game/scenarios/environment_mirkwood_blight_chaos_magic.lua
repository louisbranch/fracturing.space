local scene = Scenario.new("environment_mirkwood_blight_chaos_magic")

-- Model the chaos locus forcing double Fear dice on spellcasting.
scene:campaign{
  name = "Environment Mirkwood Blight Chaos Magic",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Gandalf")

-- Spellcasting draws extra Fear in the corrupted woods.
scene:start_session("Chaos Magic Locus")

-- Missing DSL: roll two Fear dice and take the higher on Spellcast.
scene:action_roll{ actor = "Gandalf", trait = "spellcast", difficulty = 16, outcome = "fear" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
