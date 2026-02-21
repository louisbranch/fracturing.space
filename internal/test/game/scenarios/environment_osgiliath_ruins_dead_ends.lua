local scene = Scenario.new("environment_osgiliath_ruins_dead_ends")

-- Model the Dead Ends action that shifts the city layout.
scene:campaign{
  name = "Environment Osgiliath Ruins Dead Ends",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Ghostly scenes block paths or present challenges.
scene:start_session("Dead Ends")
scene:gm_fear(1)

-- Detour/blocked-route state modeling remains unresolved.
scene:gm_spend_fear(1):spotlight("Osgiliath Ruins")

scene:end_session()

return scene
