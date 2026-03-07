local scn = Scenario.new("reaction_flow")
local dh = scn:system("DAGGERHEART")

-- Introduce Sam to highlight reaction timing.
scn:campaign{ name = "Reaction Flow", system = "DAGGERHEART", gm_mode = "HUMAN" }
scn:pc("Sam")

-- Open a session to test reaction timing.
scn:start_session("Reaction")

-- Sam makes a reaction roll under pressure.
dh:reaction{
  actor = "Sam",
  trait = "agility",
  difficulty = 8,
  outcome = "hope",
  advantage = 1,
  disadvantage = 1,
}

-- Close the session after the reaction.
scn:end_session()
return scn
