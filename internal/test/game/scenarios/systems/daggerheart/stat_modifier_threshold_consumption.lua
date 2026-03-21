local scn = Scenario.new("stat_modifier_threshold_consumption")
local dh = scn:system("DAGGERHEART")

-- Verify that stat modifiers on major_threshold and severe_threshold shift
-- damage severity tiers during damage resolution.
scn:campaign{
  name = "Stat Modifier Threshold Consumption",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "stat_modifiers"
}

-- Default thresholds: major=3, severe=6. Set explicit values for clarity.
scn:pc("Tank", { major_threshold = 4, severe_threshold = 8 })
scn:start_session("Threshold Modifiers")
dh:gm_fear(2)

-- Without modifiers: 5 damage >= major(4) but < severe(8) → major (2 marks).
dh:combined_damage{
  target = "Tank",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 5 }
  },
  expect_damage_severity = "major",
  expect_damage_marks = 2
}

-- Add +3 to major_threshold: effective major becomes 7.
-- Now 5 damage < 7 → minor (1 mark).
dh:apply_stat_modifier{
  target = "Tank",
  add = {
    { id = "mod-major-1", target = "major_threshold", delta = 3, label = "Fortified", source = "domain_card" }
  },
  source = "domain_card.fortified",
  expect_active_count = 1,
  expect_added_count = 1
}

-- Same 5 damage should now be minor because major threshold shifted to 7.
dh:combined_damage{
  target = "Tank",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 5 }
  },
  expect_damage_severity = "minor",
  expect_damage_marks = 1
}

-- Also add +4 to severe_threshold: effective severe becomes 12.
-- 9 damage: >= major(7), < severe(12) → major (2 marks).
dh:apply_stat_modifier{
  target = "Tank",
  add = {
    { id = "mod-severe-1", target = "severe_threshold", delta = 4, label = "Ironhide", source = "domain_card" }
  },
  source = "domain_card.ironhide",
  expect_active_count = 2,
  expect_added_count = 1
}

dh:combined_damage{
  target = "Tank",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 9 }
  },
  expect_damage_severity = "major",
  expect_damage_marks = 2
}

scn:end_session()

return scn
