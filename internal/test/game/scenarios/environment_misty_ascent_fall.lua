local scene = Scenario.new("environment_misty_ascent_fall")

-- Model the fall action that escalates damage by countdown state.
scene:campaign{
  name = "Environment Misty Ascent Fall",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A handhold fails, risking a deadly fall.
scene:start_session("Misty Fall")
scene:gm_fear(1)

-- Example: spend Fear, if not saved next action, damage scales by countdown.
scene:gm_spend_fear(1):spotlight("Misty Ascent")
scene:countdown_create{ name = "Fall Impact", kind = "consequence", current = 0, max = 4, direction = "increase" }
-- Damage scaling details after failed save remain unresolved.
scene:action_roll{ actor = "Frodo", trait = "agility", difficulty = 15, outcome = "failure_fear" }
scene:apply_roll_outcome{
  on_failure_fear = {
    {kind = "countdown_update", name = "Fall Impact", delta = 1, reason = "failed_save"},
  },
}

scene:end_session()

return scene
