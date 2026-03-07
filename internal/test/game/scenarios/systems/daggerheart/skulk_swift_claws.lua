local scn = Scenario.new("skulk_swift_claws")
local dh = scn:system("DAGGERHEART")

-- Model the Fell Beast's Swift Claws leap-and-strike action.
scn:campaign{
  name = "Skulk Swift Claws",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Fell Beast")

-- The wyrm marks Stress to dash in and strike.
scn:start_session("Swift Claws")

-- Example: on hit, deal 2d10+5 and force a Strength reaction to avoid knockback.
-- Partial mapping: stress-for-advantage and attack damage dice are represented.
-- Missing DSL: explicit movement and knockback branch metadata.
dh:adversary_attack{
  actor = "Fell Beast",
  target = "Frodo",
  difficulty = 0,
  stress_for_advantage = 1,
  damage_type = "physical",
  damage_dice = { { sides = 10, count = 2 } }
}

scn:end_session()

return scn
