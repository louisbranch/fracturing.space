local scn = Scenario.new("action_roll_failure_with_hope")
local dh = scn:system("DAGGERHEART")

-- Capture the failure with Hope example outcome.
scn:campaign{
  name = "Action Roll Failure with Hope",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "core"
}

scn:pc("Sam")

-- Failure still grants Hope but introduces a complication.
scn:start_session("Failure with Hope")

dh:action_roll{ actor = "Sam", trait = "agility", difficulty = 14, outcome = "failure_hope" }
dh:apply_roll_outcome{
  on_failure_hope = {
    {kind = "set_spotlight", type = "gm"},
  },
}

scn:end_session()

return scn
