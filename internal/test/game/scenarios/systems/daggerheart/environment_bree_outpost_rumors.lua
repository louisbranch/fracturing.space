local scn = Scenario.new("environment_bree_outpost_rumors")
local dh = scn:system("DAGGERHEART")

-- Capture the outpost rumors table by roll outcome.
scn:campaign{
  name = "Environment Bree Outpost Rumors",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The party asks about events and rumors.
scn:start_session("Rumors Abound")

-- Example: outcomes determine number and relevance of rumors.
-- Rumor selection fanout and failure-stress branch remain unresolved.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
