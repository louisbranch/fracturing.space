local scene = Scenario.new("environment_prancing_pony_mysterious_stranger")

-- Model the tavern action that reveals a concealed stranger.
scene:campaign{
  name = "Environment Prancing Pony Bilbo",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:npc("Bilbo")

-- A stranger reveals themselves from a shaded booth.
scene:start_session("Bilbo")
scene:gm_fear(1)

-- Example: introduce a hidden NPC without requiring a roll.
-- Narrative reveal hooks remain unresolved.
scene:gm_spend_fear(1):spotlight("Bilbo")

scene:end_session()

return scene
