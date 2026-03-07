local scn = Scenario.new("level_up_tier_entry")
local dh = scn:system("DAGGERHEART")

-- Verify tier entry at level 5 (Tier 2 to Tier 3).
scn:campaign{
  name = "Level Up Tier Entry",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "progression"
}

scn:pc("Frodo")

-- Advance through levels 2-4 to reach tier entry at level 5.
scn:start_session("Tier Entry")

-- Level 1 to 2.
dh:level_up{
  target = "Frodo",
  level_after = 2,
  advancements = { { type = "trait_increase", trait = "agility" }, { type = "add_hp_slots" } },
}

-- Level 2 to 3.
dh:level_up{
  target = "Frodo",
  level_after = 3,
  advancements = { { type = "trait_increase", trait = "strength" }, { type = "add_hp_slots" } },
}

-- Level 3 to 4.
dh:level_up{
  target = "Frodo",
  level_after = 4,
  advancements = { { type = "trait_increase", trait = "finesse" }, { type = "add_hp_slots" } },
}

-- Level 4 to 5: tier entry from T2 to T3.
-- Proficiency bump and trait marks cleared.
dh:level_up{
  target = "Frodo",
  level_after = 5,
  advancements = { { type = "increase_proficiency" } },
}

scn:end_session()

return scn
