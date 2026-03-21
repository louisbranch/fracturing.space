local scn = Scenario.new("stat_modifier_action_roll_consumption")
local dh = scn:system("DAGGERHEART")

-- Verify that base-trait stat modifiers flow into action roll totals.
-- seed=100 produces hope=7 fear=1 → base dice total = 8.
scn:campaign{
  name = "Stat Modifier Action Roll Consumption",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "stat_modifiers"
}

scn:pc("Alchemist", { evasion = 6, armor = 0 })
dh:adversary("Goblin")

scn:start_session("Trait Modifier Rolls")
dh:gm_fear(2)

-- Baseline: seed=100 produces total=8 with no modifiers.
-- Difficulty 10 → roll fails (8 < 10).
dh:action_roll{
  actor = "Alchemist",
  trait = "strength",
  difficulty = 10,
  seed = 100,
  outcome = "failure_fear",
  expect_total = 8
}

-- Apply +5 Strength modifier (Major Bolster Potion equivalent).
dh:apply_stat_modifier{
  target = "Alchemist",
  add = {
    { id = "mod-str-potion", target = "strength", delta = 5, label = "Major Bolster Potion", source = "consumable", clear_triggers = { "SHORT_REST" } }
  },
  source = "consumable.major_bolster_potion",
  expect_active_count = 1,
  expect_added_count = 1
}

-- Same seed=100, same difficulty=10 → total should be 8+5 = 13 (passes).
dh:action_roll{
  actor = "Alchemist",
  trait = "strength",
  difficulty = 10,
  seed = 100,
  outcome = "fear",
  expect_total = 13
}

-- The modifier should NOT affect a different trait.
-- seed=100 on instinct → total stays 8 (no instinct modifier applied).
dh:action_roll{
  actor = "Alchemist",
  trait = "instinct",
  difficulty = 10,
  seed = 100,
  outcome = "failure_fear",
  expect_total = 8
}

-- Short rest clears the modifier.
dh:rest{
  type = "short",
  participants = { "Alchemist" }
}

-- After rest, same seed=100 on strength → total back to 8 (fails).
dh:action_roll{
  actor = "Alchemist",
  trait = "strength",
  difficulty = 10,
  seed = 100,
  outcome = "failure_fear",
  expect_total = 8
}

scn:end_session()

return scn
