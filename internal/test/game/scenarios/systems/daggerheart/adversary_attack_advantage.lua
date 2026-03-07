local scn = Scenario.new("adversary_attack_advantage")
local dh = scn:system("DAGGERHEART")

-- Stage Saruman's ambush with a clear edge.
scn:campaign{
  name = "Adversary Attack Advantage",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "ambush"
}

scn:pc("Frodo", { armor = 1 })
dh:adversary("Saruman")

-- Saruman ambushes Frodo with a clear edge.
scn:start_session("Ambush")

-- Advantage and an attack modifier tilt the roll in Saruman's favor.
dh:adversary_attack{
  actor = "Saruman",
  target = "Frodo",
  difficulty = 0,
  attack_modifier = 2,
  advantage = 1,
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

-- Close the session after the ambush resolves.
scn:end_session()

return scn
