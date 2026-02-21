local scene = Scenario.new("environment_prancing_pony_sing")

-- Model singing for supper and its stress or reward outcomes.
scene:campaign{
  name = "Environment Prancing Pony Sing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A performance can earn coin or cost composure.
scene:start_session("Sing for Supper")

-- Example: Presence roll yields gold on success, Stress on failure.
-- Gold payout and stress branch effects remain unresolved.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "fear" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
