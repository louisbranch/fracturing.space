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
dh:adversary("Orc Stalker")

-- The shadow hides, then strikes with advantage for boosted damage.
scn:start_session("Cloaked Backstab")

-- Example: Cloaked grants Hidden; Backstab replaces damage on advantaged hit.
-- Partial mapping: Hidden application and advantaged attack are represented.
-- Missing DSL: backstab damage replacement (1d6+6) and Hidden-clear-on-attack lifecycle.
dh:apply_condition{ target = "Orc Stalker", add = { "HIDDEN" }, source = "cloaked" }
dh:adversary_attack{
  actor = "Orc Stalker",
  target = "Frodo",
  difficulty = 0,
  advantage = 1,
  damage_type = "physical",
  damage_dice = { { sides = 6, count = 1 } }
}

scn:end_session()

return scn
