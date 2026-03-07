local scn = Scenario.new("environment_bree_outpost_broken_compass")
local dh = scn:system("DAGGERHEART")

-- Capture the adventuring society passive at an outpost town.
scn:campaign{
  name = "Environment Bree Outpost Broken Compass",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
scn:npc("Elrond")

-- The society offers boasts, rumors, and rivalries.
scn:start_session("Broken Compass")
dh:gm_fear(1)

-- Example: a passive feature that sets social tension and leads.
-- Ongoing social-pressure persistence remains unresolved.
dh:gm_spend_fear(1):spotlight("Elrond")

scn:end_session()

return scn
