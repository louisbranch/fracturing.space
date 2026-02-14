local scene = Scenario.new("multi_target_attack")

-- Introduce two targets for a sweeping strike.
scene:campaign{
  name = "Multi Target Attack",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "battle"
}

scene:pc("Frodo")
scene:adversary("Nazgul")
scene:adversary("Golum")

-- Frodo swings wide to catch multiple enemies in one blow.
scene:start_session("Multi Target")

-- A single critical roll applies to both targets.
scene:multi_attack{
  actor = "Frodo",
  targets = { "Nazgul", "Golum" },
  trait = "instinct",
  difficulty = 0,
  outcome = "critical",
  expect_stress_delta = -1,
  expect_damage_total = 10,
  expect_damage_critical = true,
  expect_damage_critical_bonus = 6,
  expect_adversary_deltas = {
    { target = "Nazgul", hp_delta = -2 },
    { target = "Golum", hp_delta = -2 }
  },
  damage_type = "physical",
  damage_dice = { { sides = 6, count = 1 } }
}

-- Close the session after the sweeping strike.
scene:end_session()

return scene
