local scene = Scenario.new("help_and_resistance")

-- Frame a duel where help meets resistance.
scene:campaign{
  name = "Help and Resistance",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "help"
}

scene:pc("Frodo")
scene:adversary("Nazgul")

-- Frodo gets help to land a strike against a resistant foe.
scene:start_session("Help and Resistance")

-- Help spends hope, and resistance should blunt the physical damage.
scene:attack{
  actor = "Frodo",
  target = "Nazgul",
  trait = "presence",
  difficulty = 0,
  damage_type = "physical",
  resist_physical = true,
  modifiers = {
    Modifiers.hope("help"),
    Modifiers.mod("training", 10)
  },
  expect_hope_delta = -1,
  expect_stress_delta = 0,
  expect_damage_total = 6,
  expect_damage_critical = false,
  expect_adversary_hp_delta = -1,
  expect_adversary_damage_mitigated = true
}

-- Close the session after the assisted strike lands.
scene:end_session()

return scene
