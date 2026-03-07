local scn = Scenario.new("sweeping_attack_all_targets")
local dh = scn:system("DAGGERHEART")

-- Model the Nazgul's sweeping attack against all PCs.
scn:campaign{
  name = "Sweeping Attack All Targets",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "battle"
}

scn:pc("Sam")
scn:pc("Frodo")
scn:pc("Gandalf")
scn:pc("Aragorn")
dh:adversary("Nazgul")

-- The Nazgul spends Stress to swing at everyone within Very Close range.
scn:start_session("Sweeping Attack")

-- Example: attack total 8 compared to each PC's Evasion, misses all.
-- Missing DSL: spend adversary stress and resolve a multi-target adversary attack.
dh:adversary_attack{
  actor = "Nazgul",
  target = "Sam",
  difficulty = 0,
  damage_type = "physical"
}

-- Close the session after the sweeping attack.
scn:end_session()

return scn
