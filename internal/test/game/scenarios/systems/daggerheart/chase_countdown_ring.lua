local scn = Scenario.new("chase_countdown_ring")
local dh = scn:system("DAGGERHEART")

-- Model the ring chase with competing countdowns.
scn:campaign{
  name = "Chase Countdown Ring",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scn:pc("Sam")
scn:pc("Frodo")
dh:adversary("Golum")

-- The PCs chase a thief across a market with progress and consequence clocks.
scn:start_session("Market Chase")

dh:scene_countdown_create{ name = "PC Progress", kind = "progress", current = 0, max = 6, direction = "increase" }
dh:scene_countdown_create{ name = "Thief Escape", kind = "consequence", current = 0, max = 3, direction = "increase" }

-- Sam rolls for the chase and advances a countdown based on outcome.
dh:action_roll{ actor = "Sam", trait = "agility", difficulty = 15, total = 17 }
dh:apply_roll_outcome{
  on_success = {
    {kind = "scene_countdown_update", name = "PC Progress", delta = 1, reason = "gain_ground"},
  },
  on_failure = {
    {kind = "scene_countdown_update", name = "Thief Escape", delta = 1, reason = "escape_accelerates"},
  },
}

-- Close the session after the chase advances.
scn:end_session()

return scn
