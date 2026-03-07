local scn = Scenario.new("environment_waylaid_surprise")
local dh = scn:system("DAGGERHEART")

-- Model the Waylaid surprise action granting Fear and spotlight.
scn:campaign{
  name = "Environment Waylaid Surprise",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Orc Waylayers")

-- The ambush begins and control shifts to the attackers.
scn:start_session("Surprise")
dh:gm_fear(2)

-- Example: gain 2 Fear and immediately spotlight an ambusher.
dh:gm_spend_fear(2):spotlight("Orc Waylayers")
-- Fear-gain source remains unresolved; model immediate advantaged opening strike.
dh:adversary_attack{
  actor = "Orc Waylayers",
  target = "Frodo",
  difficulty = 0,
  advantage = 1,
  damage_type = "physical"
}

scn:end_session()

return scn
