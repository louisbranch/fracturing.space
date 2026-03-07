local scn = Scenario.new("environment_bree_outpost_rival_party")
local dh = scn:system("DAGGERHEART")

-- Model the rival party passive in an outpost town.
scn:campaign{
  name = "Environment Bree Outpost Rangers",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
scn:npc("Rangers")

-- Another adventuring party competes for the same leads.
scn:start_session("Rangers")
dh:gm_fear(1)

-- Example: establish a rival party with a personal connection.
-- Rivalry hooks and competitive-pressure persistence remain unresolved.
dh:gm_spend_fear(1):spotlight("Rangers")

scn:end_session()

return scn
