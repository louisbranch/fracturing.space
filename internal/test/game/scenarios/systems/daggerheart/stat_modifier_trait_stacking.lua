local scn = Scenario.new("stat_modifier_trait_stacking")
local dh = scn:system("DAGGERHEART")

-- Verify that multiple stat modifiers on the same base trait stack additively.
-- seed=77 produces hope=3 fear=2 → base dice total = 5.
scn:campaign{
  name = "Stat Modifier Trait Stacking",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "stat_modifiers"
}

scn:pc("Mage", { evasion = 6, armor = 0 })

scn:start_session("Trait Stacking")
dh:gm_fear(2)

-- Baseline: seed=77 produces total=5 on knowledge.
dh:action_roll{
  actor = "Mage",
  trait = "knowledge",
  difficulty = 15,
  seed = 77,
  outcome = "failure_fear",
  expect_total = 5
}

-- Apply two knowledge modifiers from different sources.
dh:apply_stat_modifier{
  target = "Mage",
  add = {
    { id = "mod-know-1", target = "knowledge", delta = 3, label = "Major Enlighten Potion", source = "consumable", clear_triggers = { "SHORT_REST" } },
    { id = "mod-know-2", target = "knowledge", delta = 4, label = "Arcane Focus", source = "domain_card", clear_triggers = { "LONG_REST" } }
  },
  source = "consumable.major_enlighten_potion",
  expect_active_count = 2,
  expect_added_count = 2
}

-- seed=77 with +3 and +4 → total = 5 + 3 + 4 = 12 (still fails difficulty 15).
dh:action_roll{
  actor = "Mage",
  trait = "knowledge",
  difficulty = 15,
  seed = 77,
  outcome = "failure_fear",
  expect_total = 12
}

-- Short rest clears only the SHORT_REST modifier (+3), leaving LONG_REST (+4).
dh:rest{
  type = "short",
  participants = { "Mage" }
}

-- seed=77 with only +4 → total = 5 + 4 = 9.
dh:action_roll{
  actor = "Mage",
  trait = "knowledge",
  difficulty = 15,
  seed = 77,
  outcome = "failure_fear",
  expect_total = 9
}

-- Long rest clears the remaining LONG_REST modifier.
dh:rest{
  type = "long",
  participants = { "Mage" }
}

-- seed=77 with no modifiers → total back to 5.
dh:action_roll{
  actor = "Mage",
  trait = "knowledge",
  difficulty = 15,
  seed = 77,
  outcome = "failure_fear",
  expect_total = 5
}

scn:end_session()

return scn
