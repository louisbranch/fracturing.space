local scn = Scenario.new("environment_prancing_pony_talk")
local dh = scn:system("DAGGERHEART")

-- Capture the tavern rumor gathering via Presence rolls.
scn:campaign{
  name = "Environment Prancing Pony Talk",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The party gathers rumors from patrons and staff.
scn:start_session("Talk of the Town")

-- Example: success grants multiple details; failure grants one plus Stress.
-- Detail fanout and stress-choice semantics remain unresolved.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
