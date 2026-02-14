local scene = Scenario.new("critical_damage")

-- Frame a duel to showcase critical damage.
scene:campaign{
  name = "Critical Damage",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "crit"
}

scene:pc("Frodo")
scene:adversary("Saruman")

-- Frodo pushes for a critical strike.
scene:start_session("Crits")

-- The roll is forced to Critical to showcase critical damage flow.
scene:attack{
  actor = "Frodo",
  target = "Saruman",
  trait = "instinct",
  difficulty = 0,
  outcome = "critical",
  expect_hope_delta = 1,
  expect_stress_delta = -1,
  expect_damage_total = 10,
  expect_damage_critical = true,
  expect_damage_critical_bonus = 6,
  expect_adversary_hp_delta = -2,
  damage_type = "physical"
}

-- Close the session after the critical blow.
scene:end_session()

return scene
