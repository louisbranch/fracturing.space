local scene = Scenario.new("damage_roll_proficiency")
local dh = scene:system("DAGGERHEART")

-- Mirror the broadsword example to emphasize proficiency-based damage dice.
scene:campaign{
  name = "Damage Roll Proficiency",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scene:pc("Frodo")
dh:adversary("Gondor Practice Dummy")

-- Frodo lands a successful hit and rolls damage using proficiency.
scene:start_session("Damage Dice")

-- Example: 2d8 for proficiency 2, rolling 3 and 7 for 10 total.
-- Missing DSL: force the damage dice to 3 and 7.
dh:damage_roll{
  actor = "Frodo",
  damage_dice = { { sides = 8, count = 2 } },
  modifier = 0
}

-- Close the session after the damage roll.
scene:end_session()

return scene
