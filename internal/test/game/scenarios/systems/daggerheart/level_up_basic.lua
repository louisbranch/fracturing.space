local scn = Scenario.new("level_up_basic")
local dh = scn:system("DAGGERHEART")

-- Verify basic level-up from level 1 to level 2.
scn:campaign{
  name = "Level Up Basic",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "progression"
}

scn:pc("Frodo")

-- Level 1 to 2: apply a trait mark advancement.
scn:start_session("Level Up")
dh:level_up{
  target = "Frodo",
  level_after = 2,
  advancements = {
    { type = "trait_increase", trait = "agility" },
  },
}

scn:end_session()

return scn
