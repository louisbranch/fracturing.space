local scn = Scenario.new("environment_bree_market_crowd_closes_in")
local dh = scn:system("DAGGERHEART")

-- Capture the crowd reaction that splits a PC from the party.
scn:campaign{
  name = "Environment Bree Market Crowd Closes In",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The crowd shifts when a PC splits off.
scn:start_session("Crowd Closes In")
dh:gm_fear(1)

-- Split-position tracking remains unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Crowd")

scn:end_session()

return scn
