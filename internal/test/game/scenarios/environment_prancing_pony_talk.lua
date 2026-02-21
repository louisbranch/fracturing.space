local scene = Scenario.new("environment_prancing_pony_talk")

-- Capture the tavern rumor gathering via Presence rolls.
scene:campaign{
  name = "Environment Prancing Pony Talk",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The party gathers rumors from patrons and staff.
scene:start_session("Talk of the Town")

-- Example: success grants multiple details; failure grants one plus Stress.
-- Detail fanout and stress-choice semantics remain unresolved.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "hope" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
