local scene = Scenario.new("armor_depletion")

-- Stage a relentless assault to grind through armor.
scene:campaign{
  name = "Armor Depletion",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scene:pc("Frodo", { armor = 2, hp = 6 })
scene:adversary("Saruman")

-- A relentless foe keeps swinging until armor gives out.
scene:start_session("Armor Depletion")

-- First hit starts chipping away at Frodo's armor.
scene:adversary_attack{
  actor = "Saruman",
  target = "Frodo",
  difficulty = 0,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_hp_delta = -1,
  expect_armor_delta = -1,
  expect_damage_total = 4,
  expect_damage_severity = "minor",
  expect_damage_marks = 1,
  expect_armor_spent = 1,
  expect_damage_mitigated = true,
  expect_damage_critical = false,
  damage_type = "physical"
}

-- Second hit should push armor closer to depletion.
scene:adversary_attack{
  actor = "Saruman",
  target = "Frodo",
  difficulty = 0,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_hp_delta = -1,
  expect_armor_delta = -1,
  expect_damage_total = 4,
  expect_damage_severity = "minor",
  expect_damage_marks = 1,
  expect_armor_spent = 1,
  expect_damage_mitigated = true,
  expect_damage_critical = false,
  damage_type = "physical"
}

-- Third hit should start eating into HP once armor is gone.
scene:adversary_attack{
  actor = "Saruman",
  target = "Frodo",
  difficulty = 0,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_hp_delta = -2,
  expect_armor_delta = 0,
  expect_damage_total = 4,
  expect_damage_severity = "major",
  expect_damage_marks = 2,
  expect_armor_spent = 0,
  expect_damage_mitigated = false,
  expect_damage_critical = false,
  damage_type = "physical"
}

-- Close the session once armor gives way.
scene:end_session()

return scene
