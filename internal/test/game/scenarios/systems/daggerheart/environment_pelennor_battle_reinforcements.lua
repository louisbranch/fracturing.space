local scene = Scenario.new("environment_pelennor_battle_reinforcements")
local dh = scene:system("DAGGERHEART")

-- Model reinforcements arriving mid-battle.
scene:campaign{
  name = "Environment Pelennor Battle Reinforcements",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Gondor Knight")
dh:adversary("Uruk-hai Minions")

-- A fresh force joins the fight.
scene:start_session("Reinforcements")
dh:gm_fear(1)

dh:adversary("Uruk-hai Vanguard")
-- Exact reinforcement composition remains unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Gondor Knight")

scene:end_session()

return scene
