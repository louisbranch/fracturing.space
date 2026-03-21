local scn = Scenario.new("stat_modifier_evasion_consumption")
local dh = scn:system("DAGGERHEART")

-- Verify that evasion stat modifiers raise the effective difficulty during
-- adversary attacks, turning hits into misses.
scn:campaign{
  name = "Stat Modifier Evasion Consumption",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "stat_modifiers"
}

-- Low evasion so the default seeded roll can hit, then miss after a modifier.
scn:pc("Rogue", { evasion = 3, armor = 0 })
dh:adversary("Goblin")

scn:start_session("Evasion Modifiers")
dh:gm_fear(2)

-- Baseline: adversary attack with difficulty=0 uses evasion(3) as difficulty.
-- Seeded roll at seed=42 should beat difficulty 3 → hit + damage.
dh:adversary_attack{
  actor = "Goblin",
  target = "Rogue",
  difficulty = 0,
  seed = 42,
  expect_hp_delta = -2,
  damage_type = "physical"
}

-- Add +100 evasion modifier to guarantee the seeded roll cannot beat it.
dh:apply_stat_modifier{
  target = "Rogue",
  add = {
    { id = "mod-evasion-wall", target = "evasion", delta = 100, label = "Wall of Iron", source = "domain_card", clear_triggers = { "SHORT_REST" } }
  },
  source = "domain_card.wall_of_iron",
  expect_active_count = 1,
  expect_added_count = 1
}

dh:adversary_attack{
  actor = "Goblin",
  target = "Rogue",
  difficulty = 0,
  seed = 42,
  expect_hp_delta = 0,
  expect_armor_delta = 0,
  damage_type = "physical"
}

-- Short rest clears the modifier; evasion returns to 3.
dh:rest{
  type = "short",
  participants = { "Rogue" }
}

-- After rest, same roll should hit again.
dh:adversary_attack{
  actor = "Goblin",
  target = "Rogue",
  difficulty = 0,
  seed = 42,
  expect_hp_delta = -2,
  damage_type = "physical"
}

scn:end_session()

return scn
