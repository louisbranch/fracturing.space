local scene = Scenario.new("environment_waylaid_surprise")

-- Model the Waylaid surprise action granting Fear and spotlight.
scene:campaign{
  name = "Environment Waylaid Surprise",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Orc Waylayers")

-- The ambush begins and control shifts to the attackers.
scene:start_session("Surprise")
scene:gm_fear(2)

-- Example: gain 2 Fear and immediately spotlight an ambusher.
scene:gm_spend_fear(2):spotlight("Orc Waylayers")
-- Fear-gain source remains unresolved; model immediate advantaged opening strike.
scene:adversary_attack{
  actor = "Orc Waylayers",
  target = "Frodo",
  difficulty = 0,
  advantage = 1,
  damage_type = "physical"
}

scene:end_session()

return scene
