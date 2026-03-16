local scn = Scenario.new("skulk_cloaked_backstab")
local dh = scn:system("DAGGERHEART")

-- Model a Skulk using Cloaked to set up a Backstab.
scn:campaign{
  name = "Skulk Cloaked Backstab",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Golum Shadow")

-- The shadow hides, then strikes with advantage for boosted damage.
scn:start_session("Cloaked Backstab")

dh:adversary_feature{
  actor = "Golum Shadow",
  feature_id = "feature.golum-cloaked"
}
dh:adversary_attack{
  actor = "Golum Shadow",
  target = "Frodo",
  feature_id = "feature.golum-backstab",
  difficulty = 0,
  damage_type = "physical"
}

scn:end_session()

return scn
