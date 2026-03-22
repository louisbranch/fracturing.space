local scn = Scenario.new("progress_countdown_climb")
local dh = scn:system("DAGGERHEART")

-- Model the mountain ascent progress countdown from the example of play.
scn:campaign{
  name = "Progress Countdown Climb",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scn:pc("Frodo")
scn:pc("Sam")

-- The GM sets a shorter progress countdown due to helpful guidance.
scn:start_session("Whitecrest Ascent")

-- Example: the ascent has three beats remaining instead of five.
dh:scene_countdown_create{
  name = "Whitecrest Ascent",
  tone = "progress",
  advancement_policy = "manual",
  fixed_starting_value = 3,
  loop_behavior = "none",
  expect_remaining_value = 3,
}

-- Sam succeeds with Fear, advancing the climb despite consequences.
-- Partial mapping: dynamic tier-based countdown advances are expressed explicitly.
dh:action_roll{ actor = "Sam", trait = "agility", difficulty = 12, outcome = "success_fear" }
dh:apply_roll_outcome{
  on_critical = {
    {kind = "scene_countdown_update", name = "Whitecrest Ascent", amount = 3, reason = "critical_ascent", expect_after_remaining = 0, expect_triggered = true},
  },
  on_success_hope = {
    {kind = "scene_countdown_update", name = "Whitecrest Ascent", amount = 2, reason = "strong_ascent", expect_after_remaining = 1},
  },
  on_success_fear = {
    {kind = "scene_countdown_update", name = "Whitecrest Ascent", amount = 1, reason = "steady_ascent", expect_after_remaining = 2},
  },
}

-- Close the session after the ascent advances.
scn:end_session()

return scn
