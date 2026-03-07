local scn = Scenario.new("countdown_variants")
local dh = scn:system("DAGGERHEART")

-- Verify countdown variant types: looping and consequence countdowns.
scn:campaign{
  name = "Countdown Variants",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdowns"
}

-- Create a looping progress countdown (variant: repeating trigger).
scn:start_session("Variants")

-- Looping countdown resets when max is reached.
dh:countdown_create{
  name = "Patrol Route",
  kind = "progress",
  current = 0,
  max = 3,
  direction = "increase",
  looping = true,
}

-- Advance the looping countdown to trigger a loop.
dh:countdown_update{ name = "Patrol Route", delta = 1, reason = "turn_1" }
dh:countdown_update{ name = "Patrol Route", delta = 1, reason = "turn_2" }
dh:countdown_update{ name = "Patrol Route", delta = 1, reason = "turn_3_loop" }

-- Create a consequence countdown (decreasing toward zero).
dh:countdown_create{
  name = "Torch Fading",
  kind = "consequence",
  current = 4,
  max = 4,
  direction = "decrease",
}

-- Tick the consequence countdown toward resolution.
dh:countdown_update{ name = "Torch Fading", delta = -1, reason = "time_passes" }
dh:countdown_update{ name = "Torch Fading", delta = -1, reason = "wind_gust" }

-- Clean up.
dh:countdown_delete{ name = "Patrol Route", reason = "scene_over" }
dh:countdown_delete{ name = "Torch Fading", reason = "relit" }

scn:end_session()

return scn
