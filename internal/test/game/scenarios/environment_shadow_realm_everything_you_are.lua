local scene = Scenario.new("environment_shadow_realm_everything_you_are")

-- Capture the looping countdown that drains a PC's highest trait.
scene:campaign{
  name = "Environment Shadow Realm Everything You Are",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The realm threatens to strip core strength away.
scene:start_session("Everything You Are")

-- Missing DSL: loop countdown and reduce highest trait or mark stress.
scene:countdown_create{ name = "Chaos Drain", kind = "loop", current = 0, max = 4, direction = "increase" }
scene:reaction_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "fear" }
scene:apply_reaction_outcome{}

scene:end_session()

return scene
