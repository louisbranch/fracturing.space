local scn = Scenario.new("skulk_reflective_scales")
local dh = scn:system("DAGGERHEART")

-- Capture the Fell Beast's reflective scales imposing disadvantage.
scn:campaign{
  name = "Skulk Reflective Scales",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Fell Beast")

-- Ranged attacks suffer disadvantage outside Very Close range.
scn:start_session("Reflective Scales")

-- Partial mapping: explicit disadvantage is represented on the attack roll.
-- Missing DSL: range-band state to gate disadvantage only beyond Very Close.
dh:attack{
  actor = "Frodo",
  target = "Fell Beast",
  trait = "instinct",
  difficulty = 0,
  disadvantage = 1,
  outcome = "hope",
  damage_type = "physical"
}

scn:end_session()

return scn
