local scn = Scenario.new("environment_misty_ascent_fall")
local dh = scn:system("DAGGERHEART")

-- Model the fall action that escalates damage by countdown state.
scn:campaign{
  name = "Environment Misty Ascent Fall",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A handhold fails, risking a deadly fall.
scn:start_session("Misty Fall")
dh:gm_fear(1)

-- Example: spend Fear, if not saved next action, damage scales by countdown.
dh:gm_spend_fear(1):spotlight("Misty Ascent")
dh:countdown_create{ name = "Fall Impact", kind = "consequence", current = 0, max = 4, direction = "increase" }
-- Damage scaling details after failed save remain unresolved.
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 15, outcome = "failure_fear" }
dh:apply_roll_outcome{
  on_failure_fear = {
    {kind = "countdown_update", name = "Fall Impact", delta = 1, reason = "failed_save"},
  },
}

scn:end_session()

return scn
