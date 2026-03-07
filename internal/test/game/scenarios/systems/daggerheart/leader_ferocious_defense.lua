local scene = Scenario.new("leader_ferocious_defense")
local dh = scene:system("DAGGERHEART")

-- Model Ferocious Defense increasing Difficulty after taking HP damage.
scene:campaign{
  name = "Leader Ferocious Defense",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
dh:adversary("Mirkwood Warden")

-- The Mirkwood Warden hardens after a damaging hit.
scene:start_session("Ferocious Defense")

-- Example: after marking HP, Difficulty increases by 1 until they mark HP.
-- Partial mapping: explicit post-hit difficulty escalation is represented.
-- Missing DSL: automatic trigger binding to qualifying HP-loss events only.
dh:attack{
  actor = "Frodo",
  target = "Mirkwood Warden",
  trait = "instinct",
  difficulty = 0,
  outcome = "hope",
  damage_type = "physical"
}
dh:adversary_update{
  target = "Mirkwood Warden",
  evasion_delta = 1,
  notes = "ferocious_defense"
}

scene:end_session()

return scene
