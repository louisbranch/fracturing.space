local scn = Scenario.new("countdown_variants")
local dh = scn:system("DAGGERHEART")

-- Verify SRD-native countdown behavior: explicit trigger resolution for loops
-- and positive advancement against remaining value.
scn:campaign{
  name = "Countdown Variants",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdowns"
}
scn:pc("Ranger")

-- Create a looping progress countdown.
scn:start_session("Variants")

-- Looping countdowns now pause in trigger_pending until the trigger is resolved.
dh:scene_countdown_create{
  name = "Patrol Route",
  tone = "progress",
  advancement_policy = "manual",
  fixed_starting_value = 3,
  loop_behavior = "reset",
  expect_starting_value = 3,
  expect_remaining_value = 3,
  expect_loop_behavior = "reset",
}

-- Advance the looping countdown to trigger, then explicitly resolve the trigger.
dh:scene_countdown_update{ name = "Patrol Route", amount = 1, reason = "turn_1", expect_remaining_value = 2, expect_after_remaining = 2 }
dh:scene_countdown_update{ name = "Patrol Route", amount = 1, reason = "turn_2", expect_remaining_value = 1, expect_after_remaining = 1 }
dh:scene_countdown_update{ name = "Patrol Route", amount = 1, reason = "turn_3_loop", expect_remaining_value = 0, expect_status = "trigger_pending", expect_triggered = true }
dh:scene_countdown_resolve_trigger{ name = "Patrol Route", reason = "restart_patrol", expect_remaining_value = 3, expect_status = "active" }

-- Create a consequence countdown with four beats of pressure remaining.
dh:scene_countdown_create{
  name = "Torch Fading",
  tone = "consequence",
  advancement_policy = "manual",
  fixed_starting_value = 4,
  loop_behavior = "none",
  expect_remaining_value = 4,
}

-- Advance the consequence countdown twice.
dh:scene_countdown_update{ name = "Torch Fading", amount = 1, reason = "time_passes", expect_remaining_value = 3, expect_after_remaining = 3 }
dh:scene_countdown_update{ name = "Torch Fading", amount = 1, reason = "wind_gust", expect_remaining_value = 2, expect_after_remaining = 2 }

-- Clean up.
dh:scene_countdown_delete{ name = "Patrol Route", reason = "scene_over" }
dh:scene_countdown_delete{ name = "Torch Fading", reason = "relit" }

scn:end_session()

return scn
