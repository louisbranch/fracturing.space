local scn = Scenario.new("damage_roll_proficiency")
local dh = scn:system("DAGGERHEART")

-- Mirror the broadsword example to emphasize proficiency-based damage dice.
scn:campaign{
  name = "Damage Roll Proficiency",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scn:pc("Frodo")
dh:adversary("Gondor Practice Dummy")

-- Frodo lands a successful hit and rolls damage using proficiency.
scn:start_session("Damage Dice")

-- Example: 2d8 for proficiency 2, rolling 3 and 7 for 10 total.
-- Missing DSL: force the damage dice to 3 and 7.
dh:damage_roll{
  actor = "Frodo",
  damage_dice = { { sides = 8, count = 2 } },
  modifier = 0
}

-- Close the session after the damage roll.
scn:end_session()

return scn
