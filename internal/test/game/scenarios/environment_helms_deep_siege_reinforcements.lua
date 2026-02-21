local scene = Scenario.new("environment_helms_deep_siege_reinforcements")

-- Capture the reinforcements action that brings in new foes.
scene:campaign{
  name = "Environment Helms Deep Siege Reinforcements",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Gondor Knight")
scene:adversary("Uruk-hai Minions")

-- New forces arrive within Far range.
scene:start_session("Reinforcements")
scene:gm_fear(1)

scene:adversary("Uruk-hai Reinforcement")
-- Party-size scaling and range placement remain unresolved in this fixture.
scene:gm_spend_fear(1):spotlight("Gondor Knight")

scene:end_session()

return scene
