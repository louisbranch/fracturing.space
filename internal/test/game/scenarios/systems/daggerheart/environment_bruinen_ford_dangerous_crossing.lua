local scn = Scenario.new("environment_bruinen_ford_dangerous_crossing")
local dh = scn:system("DAGGERHEART")

-- Model the dangerous crossing progress countdown.
scn:campaign{
  name = "Environment Bruinen Ford Dangerous Crossing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Crossing requires a progress countdown and can trigger undertow.
scn:start_session("Dangerous Crossing")

-- Example: Progress Countdown (4) with failure + Fear triggering Undertow.
dh:countdown_create{ name = "Bruinen Ford Crossing", kind = "progress", current = 0, max = 4, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 10, outcome = "failure_fear" }
dh:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "Bruinen Ford Crossing", delta = 1, reason = "crossing_progress"},
  },
  on_failure_fear = {
    {kind = "gm_spend_fear", amount = 1, target = "Bruinen Ford"},
  },
}

scn:end_session()

return scn
