local scn = Scenario.new("stat_modifier_lifecycle")
local dh = scn:system("DAGGERHEART")

-- Test the stat modifier lifecycle: add, verify, remove, rest-clearing.
scn:campaign{
  name = "Stat Modifier Lifecycle",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "stat_modifiers"
}

scn:pc("Warrior")
scn:start_session("Stat Modifiers")
dh:gm_fear(2)

-- Add an evasion modifier with short_rest clearing.
dh:apply_stat_modifier{
  target = "Warrior",
  add = {
    { id = "mod-evasion-1", target = "evasion", delta = 2, label = "Shield Wall", source = "domain_card", clear_triggers = { "SHORT_REST" } }
  },
  source = "domain_card.shield_wall",
  expect_active_count = 1,
  expect_added_count = 1
}

-- Add a second modifier (major threshold) without rest clearing.
dh:apply_stat_modifier{
  target = "Warrior",
  add = {
    { id = "mod-threshold-1", target = "major_threshold", delta = 1, label = "Fortify", source = "domain_card" }
  },
  source = "domain_card.fortify",
  expect_active_count = 2,
  expect_added_count = 1
}

-- Remove the evasion modifier explicitly.
dh:apply_stat_modifier{
  target = "Warrior",
  remove_ids = { "mod-evasion-1" },
  source = "gm_adjustment",
  expect_active_count = 1,
  expect_removed_count = 1
}

-- Re-add evasion modifier for rest-clearing test.
dh:apply_stat_modifier{
  target = "Warrior",
  add = {
    { id = "mod-evasion-2", target = "evasion", delta = 3, label = "Iron Guard", source = "domain_card", clear_triggers = { "SHORT_REST" } }
  },
  source = "domain_card.iron_guard",
  expect_active_count = 2,
  expect_added_count = 1
}

-- Short rest should clear modifiers with SHORT_REST trigger but keep others.
dh:rest{
  type = "short",
  participants = { "Warrior" }
}

-- After short rest: only the no-trigger threshold modifier survives.
-- Add two modifiers with LONG_REST clearing for long-rest test.
dh:apply_stat_modifier{
  target = "Warrior",
  add = {
    { id = "mod-prof-1", target = "proficiency", delta = 1, label = "Keen Focus", source = "domain_card", clear_triggers = { "LONG_REST" } }
  },
  source = "domain_card.keen_focus",
  expect_active_count = 2,
  expect_added_count = 1
}

-- Stack two evasion modifiers on the same stat to test stacking.
dh:apply_stat_modifier{
  target = "Warrior",
  add = {
    { id = "mod-evasion-3", target = "evasion", delta = 1, label = "Nimble", source = "domain_card", clear_triggers = { "LONG_REST" } },
    { id = "mod-evasion-4", target = "evasion", delta = 2, label = "Dodge", source = "domain_card", clear_triggers = { "LONG_REST" } }
  },
  source = "domain_card.nimble_dodge",
  expect_active_count = 4,
  expect_added_count = 2
}

-- Long rest should clear LONG_REST-triggered modifiers; the permanent threshold mod stays.
dh:rest{
  type = "long",
  participants = { "Warrior" }
}

scn:end_session()

return scn
