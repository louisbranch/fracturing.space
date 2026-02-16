local scene = Scenario.new("action_roll_failure_with_hope")

-- Capture the failure with Hope example outcome.
scene:campaign{
  name = "Action Roll Failure with Hope",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "core"
}

scene:pc("Sam")

-- Failure still grants Hope but introduces a complication.
scene:start_session("Failure with Hope")

scene:action_roll{ actor = "Sam", trait = "agility", difficulty = 14, outcome = "failure_hope" }
scene:apply_roll_outcome{
  on_failure_hope = {
    {kind = "set_spotlight", type = "gm"},
  },
}

scene:end_session()

return scene
