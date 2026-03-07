local scn = Scenario.new("environment_prancing_pony_mysterious_stranger")
local dh = scn:system("DAGGERHEART")

-- Model the tavern action that reveals a concealed stranger.
scn:campaign{
  name = "Environment Prancing Pony Bilbo",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
scn:npc("Bilbo")

-- A stranger reveals themselves from a shaded booth.
scn:start_session("Bilbo")
dh:gm_fear(1)

-- Example: introduce a hidden NPC without requiring a roll.
-- Narrative reveal hooks remain unresolved.
dh:gm_spend_fear(1):spotlight("Bilbo")

scn:end_session()

return scn
