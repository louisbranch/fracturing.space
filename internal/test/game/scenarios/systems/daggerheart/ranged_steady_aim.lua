local scn = Scenario.new("ranged_steady_aim")
local dh = scn:system("DAGGERHEART")

-- Model the Ranger of the North's Steady Aim advantage spend.
scn:campaign{
  name = "Ranged Steady Aim",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Ranger of the North")

-- The hunter marks Stress to gain advantage on their next attack.
scn:start_session("Steady Aim")

dh:adversary_attack{
  actor = "Ranger of the North",
  target = "Frodo",
  difficulty = 0,
  stress_for_advantage = 1,
  damage_type = "physical"
}

scn:end_session()

return scn
