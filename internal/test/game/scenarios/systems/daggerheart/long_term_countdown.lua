local scn = Scenario.new("long_term_countdown")
local dh = scn:system("DAGGERHEART")

-- Model a campaign countdown that advances, triggers, and resolves.
scn:campaign{
  name = "Long-Term Countdown",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scn:start_session("Campaign Clock")

dh:campaign_countdown_create{
  name = "Marius Invasion",
  tone = "progress",
  advancement_policy = "manual",
  fixed_starting_value = 2,
  loop_behavior = "reset_increase_start",
  expect_remaining_value = 2,
  expect_loop_behavior = "reset_increase_start",
}
dh:campaign_countdown_update{
  name = "Marius Invasion",
  amount = 2,
  reason = "campaign_progress",
  expect_remaining_value = 0,
  expect_status = "trigger_pending",
  expect_triggered = true,
}
dh:campaign_countdown_resolve_trigger{
  name = "Marius Invasion",
  reason = "the_next_phase_begins",
  expect_starting_value = 3,
  expect_remaining_value = 3,
  expect_status = "active",
}
dh:campaign_countdown_delete{ name = "Marius Invasion", reason = "threat_reframed" }

-- Close the session after advancing the long-term clock.
scn:end_session()

return scn
