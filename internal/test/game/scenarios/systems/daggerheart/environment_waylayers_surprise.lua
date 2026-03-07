local scene = Scenario.new("environment_waylayers_surprise")
local dh = scene:system("DAGGERHEART")

-- Model the ambushers' surprise action for a sudden strike.
scene:campaign{
  name = "Environment Orc Waylayers Surprise",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Orc Waylayers")

-- The ambush begins, shifting the spotlight and adding Fear.
scene:start_session("Ambush")
dh:gm_fear(2)

-- Example: Surprise grants 2 Fear and spotlights an ambusher.
dh:gm_spend_fear(2):spotlight("Orc Waylayers")
-- Fear-gain source remains unresolved; model immediate advantaged opening strike.
dh:adversary_attack{
  actor = "Orc Waylayers",
  target = "Frodo",
  difficulty = 0,
  advantage = 1,
  damage_type = "physical"
}

scene:end_session()

return scene
