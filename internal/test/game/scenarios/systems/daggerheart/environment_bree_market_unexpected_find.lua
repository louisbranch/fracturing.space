local scn = Scenario.new("environment_bree_market_unexpected_find")
local dh = scn:system("DAGGERHEART")

-- Model the marketplace action that reveals a needed item.
scn:campaign{
  name = "Environment Bree Market Unexpected Find",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
scn:npc("Bilbo")

-- A merchant reveals a rare or desired item.
scn:start_session("Unexpected Find")
dh:gm_fear(1)

-- Quest-item discovery is a narrative event, not a gold mechanic.
-- The GM spends fear to reveal the item to an NPC merchant.
dh:gm_spend_fear(1):spotlight("Bilbo")

scn:end_session()

return scn
