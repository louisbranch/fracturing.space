local scene = Scenario.new("countdown_lifecycle")

scene:campaign{ name = "Countdown Lifecycle", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:start_session("Countdowns")

scene:countdown_create{ name = "Doom", kind = "progress", current = 0, max = 4, direction = "increase" }
scene:countdown_update{ name = "Doom", delta = 1, reason = "tick" }
scene:countdown_delete{ name = "Doom", reason = "resolved" }

scene:end_session()
return scene
