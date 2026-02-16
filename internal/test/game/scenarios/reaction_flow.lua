local scene = Scenario.new("reaction_flow")

-- Introduce Sam to highlight reaction timing.
scene:campaign{ name = "Reaction Flow", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Sam")

-- Open a session to test reaction timing.
scene:start_session("Reaction")

-- Sam makes a reaction roll under pressure.
scene:reaction{
  actor = "Sam",
  trait = "agility",
  difficulty = 8,
  outcome = "hope",
  advantage = 1,
  disadvantage = 1,
}

-- Close the session after the reaction.
scene:end_session()
return scene
