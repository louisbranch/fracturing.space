local scn = Scenario.new("combined_damage_sources")
local dh = scn:system("DAGGERHEART")

-- Introduce two attackers to combine their damage against Bilbo.
scn:campaign{
  name = "Combined Damage Sources",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scn:pc("Frodo")
scn:pc("Sam")
dh:adversary("Bilbo")

-- Frodo and Sam land separate hits that combine into one damage total.
scn:start_session("Combined Damage")

-- Their damage is summed before comparing against thresholds.
dh:combined_damage{
  target = "Bilbo",
  damage_type = "physical",
  expect_adversary_hp_delta = -3,
  expect_adversary_armor_delta = 0,
  expect_damage_severity = "severe",
  expect_damage_marks = 3,
  expect_armor_spent = 0,
  expect_damage_mitigated = false,
  sources = {
    { character = "Frodo", amount = 6 },
    { character = "Sam", amount = 6 }
  }
}

-- Close the session after the combined damage check.
scn:end_session()

return scn
