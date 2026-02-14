local scene = Scenario.new("armor_mitigation")

-- Frame a duel where armor mitigation should matter.
scene:campaign{
  name = "Armor Mitigation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scene:pc("Frodo", { armor = 1 })
scene:adversary("Nazgul")

-- Nazgul pressures Frodo while the GM holds fear to power the assault.
scene:start_session("Armor")
scene:gm_fear(2)

-- Nazgul lands a hit; Frodo is expected to mitigate with armor.
scene:adversary_attack{
  actor = "Nazgul",
  target = "Frodo",
  difficulty = 0,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_hp_delta = -2,
  expect_armor_delta = -1,
  expect_damage_total = 4,
  expect_damage_severity = "major",
  expect_damage_marks = 2,
  expect_armor_spent = 1,
  expect_damage_mitigated = true,
  expect_damage_critical = false,
  damage_type = "physical"
}

-- Close the session after the mitigation check.
scene:end_session()

return scene
