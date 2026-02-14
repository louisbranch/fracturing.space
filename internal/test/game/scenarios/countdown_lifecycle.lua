local scene = Scenario.new("countdown_lifecycle")

-- Open a campaign focused on a single progress countdown.
scene:campaign{ name = "Countdown Lifecycle", system = "DAGGERHEART", gm_mode = "HUMAN" }

-- Kick off a session to exercise countdowns.
scene:start_session("Countdowns")

-- Create, advance, and resolve a single countdown.
scene:countdown_create{ name = "Doom", kind = "progress", current = 0, max = 4, direction = "increase" }
scene:countdown_update{ name = "Doom", delta = 1, reason = "tick" }
scene:countdown_delete{ name = "Doom", reason = "resolved" }

-- Close the session once the countdown resolves.
scene:end_session()
return scene
