local scn = Scenario.new("environment_pelennor_battle_reinforcements")
local dh = scn:system("DAGGERHEART")

-- Model reinforcements arriving mid-battle.
scn:campaign{
  name = "Environment Pelennor Battle Reinforcements",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Gondor Knight")
dh:adversary("Uruk-hai Minions")

-- A fresh force joins the fight.
scn:start_session("Reinforcements")
dh:gm_fear(1)

dh:adversary("Uruk-hai Vanguard")
-- Exact reinforcement composition remains unresolved in this fixture.
dh:gm_spend_fear(1):spotlight("Gondor Knight")

scn:end_session()

return scn
