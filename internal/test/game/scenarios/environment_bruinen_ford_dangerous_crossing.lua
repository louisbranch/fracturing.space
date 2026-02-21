local scene = Scenario.new("environment_bruinen_ford_dangerous_crossing")

-- Model the dangerous crossing progress countdown.
scene:campaign{
  name = "Environment Bruinen Ford Dangerous Crossing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Crossing requires a progress countdown and can trigger undertow.
scene:start_session("Dangerous Crossing")

-- Example: Progress Countdown (4) with failure + Fear triggering Undertow.
scene:countdown_create{ name = "Bruinen Ford Crossing", kind = "progress", current = 0, max = 4, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "agility", difficulty = 10, outcome = "failure_fear" }
scene:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "Bruinen Ford Crossing", delta = 1, reason = "crossing_progress"},
  },
  on_failure_fear = {
    {kind = "gm_spend_fear", amount = 1, target = "Bruinen Ford"},
  },
}

scene:end_session()

return scene
