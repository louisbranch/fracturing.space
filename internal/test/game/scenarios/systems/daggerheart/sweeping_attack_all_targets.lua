local scene = Scenario.new("sweeping_attack_all_targets")
local dh = scene:system("DAGGERHEART")

-- Model the Nazgul's sweeping attack against all PCs.
scene:campaign{
  name = "Sweeping Attack All Targets",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "battle"
}

scene:pc("Sam")
scene:pc("Frodo")
scene:pc("Gandalf")
scene:pc("Aragorn")
dh:adversary("Nazgul")

-- The Nazgul spends Stress to swing at everyone within Very Close range.
scene:start_session("Sweeping Attack")

-- Example: attack total 8 compared to each PC's Evasion, misses all.
-- Missing DSL: spend adversary stress and resolve a multi-target adversary attack.
dh:adversary_attack{
  actor = "Nazgul",
  target = "Sam",
  difficulty = 0,
  damage_type = "physical"
}

-- Close the session after the sweeping attack.
scene:end_session()

return scene
