local scene = Scenario.new("environment_dark_tower_usurpation_defilers_abound")
local dh = scene:system("DAGGERHEART")

-- Capture summoning Orc Shock Troops and their group attack.
scene:campaign{
  name = "Environment Dark Tower Usurpation Defilers Abound",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Orc Shock Troops")

-- The usurper calls in shock troops.
scene:start_session("Defilers Abound")
dh:gm_fear(2)

dh:adversary("Orc Shock Troops Reinforcement 1")
dh:adversary("Orc Shock Troops Reinforcement 2")
-- Variable summon count and immediate group-attack execution remain unresolved.
dh:gm_spend_fear(2):spotlight("Orc Shock Troops Reinforcement 1")

scene:end_session()

return scene
