local scn = Scenario.new("countdown_linked_pair")
local dh = scn:system("DAGGERHEART")

-- Model linked progress and consequence countdowns moving through one roll outcome.
scn:campaign{
  name = "Countdown Linked Pair",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scn:pc("Scout")

scn:start_session("Floodgate")

dh:scene_countdown_create{
  name = "Seal the Gate",
  tone = "progress",
  advancement_policy = "manual",
  fixed_starting_value = 4,
  loop_behavior = "none",
  expect_starting_value = 4,
  expect_remaining_value = 4,
}

dh:scene_countdown_create{
  name = "Rising Flood",
  tone = "consequence",
  advancement_policy = "manual",
  fixed_starting_value = 3,
  loop_behavior = "none",
  linked_countdown_id = "Seal the Gate",
  expect_remaining_value = 3,
  expect_linked_countdown_id = "Seal the Gate",
}

dh:action_roll{ actor = "Scout", trait = "agility", difficulty = 12, outcome = "success_fear" }
dh:apply_roll_outcome{
  on_success_fear = {
    {kind = "scene_countdown_update", name = "Seal the Gate", amount = 1, reason = "gain_ground", expect_after_remaining = 3},
    {kind = "scene_countdown_update", name = "Rising Flood", amount = 1, reason = "pressure_mounts", expect_after_remaining = 2},
  },
}

scn:end_session()

return scn
