local scn = Scenario.new("armor_depletion")
local dh = scn:system("DAGGERHEART")

-- Stage a relentless assault to grind through armor.
scn:campaign{
  name = "Armor Depletion",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo", { armor = 2, hp = 6 })
dh:adversary("Saruman")

-- A relentless foe keeps swinging until armor gives out.
scn:start_session("Armor Depletion")

-- First hit starts chipping away at Frodo's armor.
dh:adversary_attack{
  actor = "Saruman",
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

-- Second hit should push armor closer to depletion.
dh:adversary_attack{
  actor = "Saruman",
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

-- Third hit should start eating into HP once armor is gone.
dh:adversary_attack{
  actor = "Saruman",
  target = "Frodo",
  difficulty = 0,
  expect_hope_delta = 0,
  expect_stress_delta = 0,
  expect_hp_delta = -2,
  expect_armor_delta = 0,
  expect_damage_total = 4,
  expect_damage_severity = "severe",
  expect_damage_marks = 3,
  expect_armor_spent = 0,
  expect_damage_mitigated = false,
  expect_damage_critical = false,
  damage_type = "physical"
}

-- Close the session once armor gives way.
scn:end_session()

return scn
