local scn = Scenario.new("countdown_randomized_start")
local dh = scn:system("DAGGERHEART")

-- Model a countdown that uses the randomized-start path with deterministic replay RNG.
scn:campaign{
  name = "Countdown Randomized Start",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scn:pc("Guide")

scn:start_session("Sands")

dh:scene_countdown_create{
  name = "Shifting Sands",
  tone = "consequence",
  advancement_policy = "manual",
  randomized_start = { min = 3, max = 3, seed = 17 },
  loop_behavior = "none",
  expect_starting_value = 3,
  expect_remaining_value = 3,
  expect_status = "active",
  expect_starting_roll_min = 3,
  expect_starting_roll_max = 3,
  expect_starting_roll_value = 3,
}

dh:scene_countdown_update{
  name = "Shifting Sands",
  amount = 1,
  reason = "the_dunes_shift",
  expect_remaining_value = 2,
  expect_after_remaining = 2,
}

scn:end_session()

return scn
