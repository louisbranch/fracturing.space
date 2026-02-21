local scene = Scenario.new("orc_archer_opportunist")

-- Highlight damage doubling from the Opportunist feature.
scene:campaign{
  name = "Orc Archer Opportunist",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scene:pc("Frodo", { armor = 1 })
scene:adversary("Orc Archer")
scene:adversary("Orc Raider")

-- The archer attacks while allies crowd the target.
scene:start_session("Opportunist Shot")

-- Example: 1d8+1 damage is doubled when multiple foes are Very Close.
-- Partial mapping: trigger precondition marker plus doubled strike are explicit.
-- Missing DSL: spatial verification for Very Close range before the passive triggers.
scene:adversary_update{ target = "Orc Archer", notes = "opportunist_two_nearby_adversaries" }
scene:adversary_attack{
  actor = "Orc Archer",
  target = "Frodo",
  difficulty = 0,
  attack_modifier = 2,
  damage_dice = {{count = 2, sides = 8}},
  damage_type = "physical"
}
scene:set_spotlight{ target = "Frodo" }

-- Close the session after the opportunist shot.
scene:end_session()

return scene
