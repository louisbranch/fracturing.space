local scene = Scenario.new("chase_countdown_ring")

-- Model the ring chase with competing countdowns.
scene:campaign{
  name = "Chase Countdown Ring",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scene:pc("Sam")
scene:pc("Frodo")
scene:adversary("Golum")

-- The PCs chase a thief across a market with progress and consequence clocks.
scene:start_session("Market Chase")

scene:countdown_create{ name = "PC Progress", kind = "progress", current = 0, max = 6, direction = "increase" }
scene:countdown_create{ name = "Thief Escape", kind = "consequence", current = 0, max = 3, direction = "increase" }

-- Sam rolls for the chase and advances a countdown based on outcome.
scene:action_roll{ actor = "Sam", trait = "agility", difficulty = 15, total = 17 }
scene:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "PC Progress", delta = 1, reason = "gain_ground"},
  },
  on_failure = {
    {kind = "countdown_update", name = "Thief Escape", delta = 1, reason = "escape_accelerates"},
  },
}

-- Close the session after the chase advances.
scene:end_session()

return scene
