local scene = Scenario.new("skulk_swift_claws")

-- Model the Fell Beast's Swift Claws leap-and-strike action.
scene:campaign{
  name = "Skulk Swift Claws",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
scene:adversary("Fell Beast")

-- The wyrm marks Stress to dash in and strike.
scene:start_session("Swift Claws")

-- Example: on hit, deal 2d10+5 and force a Strength reaction to avoid knockback.
-- Partial mapping: stress-for-advantage and attack damage dice are represented.
-- Missing DSL: explicit movement and knockback branch metadata.
scene:adversary_attack{
  actor = "Fell Beast",
  target = "Frodo",
  difficulty = 0,
  stress_for_advantage = 1,
  damage_type = "physical",
  damage_dice = { { sides = 10, count = 2 } }
}

scene:end_session()

return scene
