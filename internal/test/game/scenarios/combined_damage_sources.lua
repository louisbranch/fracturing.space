local scene = Scenario.new("combined_damage_sources")

-- Introduce two attackers to combine their damage against Bilbo.
scene:campaign{
  name = "Combined Damage Sources",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scene:pc("Frodo")
scene:pc("Sam")
scene:adversary("Bilbo")

-- Frodo and Sam land separate hits that combine into one damage total.
scene:start_session("Combined Damage")

-- Their damage is summed before comparing against thresholds.
scene:combined_damage{
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
scene:end_session()

return scene
