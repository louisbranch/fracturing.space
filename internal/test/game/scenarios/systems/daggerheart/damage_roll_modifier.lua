local scn = Scenario.new("damage_roll_modifier")
local dh = scn:system("DAGGERHEART")

-- Reflect the shortbow example to show modifiers on damage rolls.
scn:campaign{
  name = "Damage Roll Modifier",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scn:pc("Sam")
dh:adversary("Orc Raider")

-- Sam hits with a weapon that adds a flat modifier.
scn:start_session("Damage Modifier")

-- Example: 3d6 (3, 5, 6) + 6 for 20 total physical damage.
-- Missing DSL: force the damage dice to 3, 5, and 6.
dh:damage_roll{
  actor = "Sam",
  damage_dice = { { sides = 6, count = 3 } },
  modifier = 6
}

-- Close the session after the damage roll.
scn:end_session()

return scn
