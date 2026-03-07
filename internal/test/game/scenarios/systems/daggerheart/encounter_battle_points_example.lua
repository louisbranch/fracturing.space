local scn = Scenario.new("encounter_battle_points_example")
local dh = scn:system("DAGGERHEART")

-- Summarize the battle point budgeting example for encounter prep.
-- Clarification-gated fixture (P31): encounter budgeting remains prep-time guidance.
scn:campaign{
  name = "Encounter Battle Points Example",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "encounter"
}

-- The GM budgets an easier encounter for four PCs.
scn:start_session("Battle Points")

-- Example: 4 PCs = 14 points, adjust to 13, spend on 2 Bruisers, 2 Standards, 4 Minions.
-- Missing DSL: encode battle point budgeting and encounter composition.

scn:end_session()

return scn
