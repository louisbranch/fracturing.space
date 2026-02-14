local scene = Scenario.new("action_roll_outcomes")

-- Introduce Frodo and three foes to showcase roll outcomes.
scene:campaign{
  name = "Action Roll Outcomes",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "outcomes"
}

scene:pc("Frodo")
scene:adversary("Nazgul")
scene:adversary("Golum")
scene:adversary("Saruman")

-- Frodo tests three distinct action roll outcomes in a single session.
scene:start_session("Outcomes")

-- Hope outcome: the player gains momentum from a clean success.
-- Missing DSL: assert hope increases or any follow-up advantage.
scene:attack{
  actor = "Frodo",
  target = "Nazgul",
  trait = "instinct",
  difficulty = 0,
  outcome = "hope",
  damage_type = "physical"
}

-- Fear outcome: the GM seizes the moment after the roll.
-- Missing DSL: assert fear increases or spotlight shifts.
scene:attack{
  actor = "Frodo",
  target = "Golum",
  trait = "instinct",
  difficulty = 0,
  outcome = "fear",
  damage_type = "physical"
}

-- Critical outcome: the roll spikes to an exceptional result.
-- Missing DSL: assert crit damage or other critical effects.
scene:attack{
  actor = "Frodo",
  target = "Saruman",
  trait = "instinct",
  difficulty = 0,
  outcome = "critical",
  damage_type = "physical"
}

-- Close the session after all outcomes land.
scene:end_session()

return scene
