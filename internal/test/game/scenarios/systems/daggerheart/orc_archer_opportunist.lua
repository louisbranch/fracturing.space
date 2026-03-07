local scn = Scenario.new("orc_archer_opportunist")
local dh = scn:system("DAGGERHEART")

-- Highlight damage doubling from the Opportunist feature.
scn:campaign{
  name = "Orc Archer Opportunist",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scn:pc("Frodo", { armor = 1 })
dh:adversary("Orc Archer")
dh:adversary("Orc Raider")

-- The archer attacks while allies crowd the target.
scn:start_session("Opportunist Shot")

-- Example: 1d8+1 damage is doubled when multiple foes are Very Close.
-- Partial mapping: trigger precondition marker plus doubled strike are explicit.
-- Missing DSL: spatial verification for Very Close range before the passive triggers.
dh:adversary_update{ target = "Orc Archer", notes = "opportunist_two_nearby_adversaries" }
dh:adversary_attack{
  actor = "Orc Archer",
  target = "Frodo",
  difficulty = 0,
  attack_modifier = 2,
  damage_dice = {{count = 2, sides = 8}},
  damage_type = "physical"
}
scn:set_spotlight{ target = "Frodo" }

-- Close the session after the opportunist shot.
scn:end_session()

return scn
