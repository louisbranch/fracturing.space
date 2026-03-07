local scene = Scenario.new("skulk_reflective_scales")
local dh = scene:system("DAGGERHEART")

-- Capture the Fell Beast's reflective scales imposing disadvantage.
scene:campaign{
  name = "Skulk Reflective Scales",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
dh:adversary("Fell Beast")

-- Ranged attacks suffer disadvantage outside Very Close range.
scene:start_session("Reflective Scales")

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

scene:end_session()

return scene
