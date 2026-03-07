local scn = Scenario.new("armor_mitigation")
local dh = scn:system("DAGGERHEART")

-- Frame a duel where armor mitigation should matter.
scn:campaign{
  name = "Armor Mitigation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo", { armor = 1 })
dh:adversary("Nazgul")

-- Nazgul pressures Frodo while the GM holds fear to power the assault.
scn:start_session("Armor")
dh:gm_fear(2)

-- Nazgul lands a hit; Frodo is expected to mitigate with armor.
dh:adversary_attack{
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
scn:end_session()

return scn
