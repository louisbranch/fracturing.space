local scn = Scenario.new("long_term_countdown")
local dh = scn:system("DAGGERHEART")

-- Model the long-term countdown example for a growing invasion.
scn:campaign{
  name = "Long-Term Countdown",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

-- The GM tracks Marius's invasion over several sessions.
scn:start_session("Long-Term Clock")

-- Example: a long-term countdown set to 8 ticks.
dh:countdown_create{ name = "Marius Invasion", kind = "long_term", current = 0, max = 8, direction = "increase" }
dh:countdown_update{ name = "Marius Invasion", delta = 1, reason = "campaign_progress" }
dh:countdown_update{ name = "Marius Invasion", delta = 1, reason = "session_end" }

-- Close the session after advancing the long-term clock.
scn:end_session()

return scn
