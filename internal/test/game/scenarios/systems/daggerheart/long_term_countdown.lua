local scene = Scenario.new("long_term_countdown")
local dh = scene:system("DAGGERHEART")

-- Model the long-term countdown example for a growing invasion.
scene:campaign{
  name = "Long-Term Countdown",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

-- The GM tracks Marius's invasion over several sessions.
scene:start_session("Long-Term Clock")

-- Example: a long-term countdown set to 8 ticks.
dh:countdown_create{ name = "Marius Invasion", kind = "long_term", current = 0, max = 8, direction = "increase" }
dh:countdown_update{ name = "Marius Invasion", delta = 1, reason = "campaign_progress" }
dh:countdown_update{ name = "Marius Invasion", delta = 1, reason = "session_end" }

-- Close the session after advancing the long-term clock.
scene:end_session()

return scene
