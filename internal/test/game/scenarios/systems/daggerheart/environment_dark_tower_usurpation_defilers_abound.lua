local scn = Scenario.new("environment_dark_tower_usurpation_defilers_abound")
local dh = scn:system("DAGGERHEART")

-- Capture summoning Orc Shock Troops and their group attack.
scn:campaign{
  name = "Environment Dark Tower Usurpation Defilers Abound",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Orc Shock Troops")

-- The usurper calls in shock troops.
scn:start_session("Defilers Abound")
dh:gm_fear(2)

dh:adversary("Orc Shock Troops Reinforcement 1")
dh:adversary("Orc Shock Troops Reinforcement 2")
-- Variable summon count and immediate group-attack execution remain unresolved.
dh:gm_spend_fear(2):spotlight("Orc Shock Troops Reinforcement 1")

scn:end_session()

return scn
