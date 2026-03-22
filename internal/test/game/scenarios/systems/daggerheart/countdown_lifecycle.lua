local scn = Scenario.new("countdown_lifecycle")
local dh = scn:system("DAGGERHEART")

-- Open a campaign focused on a single scene countdown lifecycle.
scn:campaign{ name = "Countdown Lifecycle", system = "DAGGERHEART", gm_mode = "HUMAN" }
scn:pc("Scout")

-- Kick off a session to exercise countdown creation, trigger, and resolution.
scn:start_session("Countdowns")

dh:scene_countdown_create{
  name = "Doom",
  tone = "progress",
  advancement_policy = "manual",
  fixed_starting_value = 2,
  loop_behavior = "none",
  expect_starting_value = 2,
  expect_remaining_value = 2,
  expect_status = "active",
}
dh:scene_countdown_update{
  name = "Doom",
  amount = 1,
  reason = "first_tick",
  expect_remaining_value = 1,
  expect_before_remaining = 2,
  expect_after_remaining = 1,
  expect_advanced_by = 1,
  expect_triggered = false,
}
dh:scene_countdown_update{
  name = "Doom",
  amount = 1,
  reason = "final_tick",
  expect_remaining_value = 0,
  expect_status = "trigger_pending",
  expect_before_remaining = 1,
  expect_after_remaining = 0,
  expect_advanced_by = 1,
  expect_triggered = true,
}
dh:scene_countdown_resolve_trigger{
  name = "Doom",
  reason = "countdown_effect_resolved",
  expect_remaining_value = 0,
  expect_status = "active",
}
dh:scene_countdown_delete{ name = "Doom", reason = "resolved" }

-- Close the session once the countdown resolves.
scn:end_session()
return scn
