local scene = Scenario.new("action_roll_outcomes")

-- Introduce Frodo and three foes to showcase roll outcomes.
scene:campaign{
  name = "Action Roll Outcomes",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "outcomes"
}

scene:pc("Frodo", { stress = 1 })
scene:adversary("Nazgul")
scene:adversary("Golum")
scene:adversary("Saruman")

-- Frodo tests three distinct action roll outcomes in a single session.
scene:start_session("Outcomes")

-- Hope outcome: the player gains momentum from a clean success.
scene:attack{
  actor = "Frodo",
  target = "Nazgul",
  trait = "instinct",
  difficulty = 0,
  outcome = "hope",
  expect_outcome = "hope",
  expect_hope_delta = 1,
  expect_damage_total = 5,
  expect_damage_critical = false,
  expect_adversary_hp_delta = -1,
  damage_type = "physical"
}

-- Fear outcome: the GM seizes the moment after the roll.
scene:attack{
  actor = "Frodo",
  target = "Golum",
  trait = "instinct",
  difficulty = 0,
  outcome = "fear",
  expect_outcome = "fear",
  expect_gm_fear_delta = 1,
  expect_spotlight = "gm",
  expect_requires_complication = true,
  expect_damage_total = 2,
  expect_damage_critical = false,
  expect_adversary_hp_delta = -1,
  damage_type = "physical"
}

-- Critical outcome: the roll spikes to an exceptional result.
scene:attack{
  actor = "Frodo",
  target = "Saruman",
  trait = "instinct",
  difficulty = 0,
  outcome = "critical",
  expect_outcome = "critical",
  expect_hope_delta = 1,
  expect_stress_delta = -1,
  expect_damage_total = 10,
  expect_damage_critical = true,
  expect_damage_critical_bonus = 6,
  expect_adversary_hp_delta = -2,
  damage_type = "physical"
}

-- Close the session after all outcomes land.
scene:end_session()

return scene
