local scene = Scenario.new("skulk_icicle_barb")
local dh = scene:system("DAGGERHEART")

-- Capture the Fell Beast's Icicle Barb group attack.
scene:campaign{
  name = "Skulk Icicle Barb",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
scene:pc("Sam")
dh:adversary("Fell Beast")

-- The wyrm spends Fear to pin a group with barbs.
scene:start_session("Icicle Barb")
dh:gm_fear(1)

-- Example: targets take 2d4 damage and become Restrained until they break free.
-- Partial mapping: per-target adversary attacks + Restrained condition application.
-- Missing DSL: single shared-roll group-attack branching across all targets.
dh:gm_spend_fear(1):spotlight("Fell Beast")
dh:adversary_attack{
  actor = "Fell Beast",
  target = "Frodo",
  difficulty = 0,
  seed = 41,
  damage_type = "physical",
  damage_dice = { { sides = 4, count = 2 } }
}
dh:adversary_attack{
  actor = "Fell Beast",
  target = "Sam",
  difficulty = 0,
  seed = 41,
  damage_type = "physical",
  damage_dice = { { sides = 4, count = 2 } }
}
dh:apply_condition{ target = "Frodo", add = { "RESTRAINED" }, source = "icicle_barb" }
dh:apply_condition{ target = "Sam", add = { "RESTRAINED" }, source = "icicle_barb" }

scene:end_session()

return scene
