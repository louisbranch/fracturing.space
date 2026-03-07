local scn = Scenario.new("sam_critical_broadsword")
local dh = scn:system("DAGGERHEART")

-- Reflect Sam's critical broadsword strike with advantage and reroll.
scn:campaign{
  name = "Sam Critical Broadsword",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "crit"
}

scn:pc("Sam")
dh:adversary("Nazgul")

-- Sam marks stress for advantage and scores a critical.
scn:start_session("Critical Strike")

-- Example: critical success adds max damage dice before rolling 2d8 +2.
-- Partial mapping: attack roll includes declared advantage.
-- Missing DSL/API: evented stress spend and reroll-replacement mechanics.
dh:attack{
  actor = "Sam",
  target = "Nazgul",
  trait = "agility",
  difficulty = 0,
  advantage = 1,
  outcome = "critical",
  damage_type = "physical"
}
dh:damage_roll{
  actor = "Sam",
  damage_dice = { { sides = 8, count = 2 } },
  modifier = 2
}

-- Close the session after the critical hit.
scn:end_session()

return scn
