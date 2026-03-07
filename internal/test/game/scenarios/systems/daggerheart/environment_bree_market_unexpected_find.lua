local scene = Scenario.new("environment_bree_market_unexpected_find")
local dh = scene:system("DAGGERHEART")

-- Model the marketplace action that reveals a needed item.
scene:campaign{
  name = "Environment Bree Market Unexpected Find",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:npc("Bilbo")

-- A merchant reveals a rare or desired item.
scene:start_session("Unexpected Find")
dh:gm_fear(1)

-- Quest-item payload and non-gold cost semantics remain unresolved.
dh:gm_spend_fear(1):spotlight("Bilbo")

scene:end_session()

return scene
