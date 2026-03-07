local scn = Scenario.new("countdown_lifecycle")
local dh = scn:system("DAGGERHEART")

-- Open a campaign focused on a single progress countdown.
scn:campaign{ name = "Countdown Lifecycle", system = "DAGGERHEART", gm_mode = "HUMAN" }

-- Kick off a session to exercise countdowns.
scn:start_session("Countdowns")

-- Create, advance, and resolve a single countdown.
dh:countdown_create{ name = "Doom", kind = "progress", current = 0, max = 4, direction = "increase" }
dh:countdown_update{ name = "Doom", delta = 1, reason = "tick" }
dh:countdown_delete{ name = "Doom", reason = "resolved" }

-- Close the session once the countdown resolves.
scn:end_session()
return scn
