local scn = Scenario.new("environment_helms_deep_siege_reinforcements")
local dh = scn:system("DAGGERHEART")

-- Capture the reinforcements action that brings in new foes.
scn:campaign{
  name = "Environment Helms Deep Siege Reinforcements",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Gondor Knight")
dh:adversary("Uruk-hai Minions")

-- New forces arrive within Far range.
scn:start_session("Reinforcements")
dh:gm_fear(1)

dh:adversary("Uruk-hai Reinforcement")
-- Party-size scaling and range placement remain unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Gondor Knight")

scn:end_session()

return scn
