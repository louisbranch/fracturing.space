local scene = Scenario.new("environment_bree_outpost_rumors")
local dh = scene:system("DAGGERHEART")

-- Capture the outpost rumors table by roll outcome.
scene:campaign{
  name = "Environment Bree Outpost Rumors",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The party asks about events and rumors.
scene:start_session("Rumors Abound")

-- Example: outcomes determine number and relevance of rumors.
-- Rumor selection fanout and failure-stress branch remain unresolved.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 12, outcome = "hope" }
dh:apply_roll_outcome{}

scene:end_session()

return scene
