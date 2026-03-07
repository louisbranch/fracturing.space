local scn = Scenario.new("environment_osgiliath_ruins_dead_ends")
local dh = scn:system("DAGGERHEART")

-- Model the Dead Ends action that shifts the city layout.
scn:campaign{
  name = "Environment Osgiliath Ruins Dead Ends",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Ghostly scenes block paths or present challenges.
scn:start_session("Dead Ends")
dh:gm_fear(1)

-- Detour/blocked-route state modeling remains unresolved.
dh:gm_spend_fear(1):spotlight("Osgiliath Ruins")

scn:end_session()

return scn
