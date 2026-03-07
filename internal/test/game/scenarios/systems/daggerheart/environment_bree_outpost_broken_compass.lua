local scene = Scenario.new("environment_bree_outpost_broken_compass")
local dh = scene:system("DAGGERHEART")

-- Capture the adventuring society passive at an outpost town.
scene:campaign{
  name = "Environment Bree Outpost Broken Compass",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:npc("Elrond")

-- The society offers boasts, rumors, and rivalries.
scene:start_session("Broken Compass")
dh:gm_fear(1)

-- Example: a passive feature that sets social tension and leads.
-- Ongoing social-pressure persistence remains unresolved.
dh:gm_spend_fear(1):spotlight("Elrond")

scene:end_session()

return scene
