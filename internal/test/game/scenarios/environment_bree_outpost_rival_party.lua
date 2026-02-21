local scene = Scenario.new("environment_bree_outpost_rival_party")

-- Model the rival party passive in an outpost town.
scene:campaign{
  name = "Environment Bree Outpost Rangers",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:npc("Rangers")

-- Another adventuring party competes for the same leads.
scene:start_session("Rangers")
scene:gm_fear(1)

-- Example: establish a rival party with a personal connection.
-- Rivalry hooks and competitive-pressure persistence remain unresolved.
scene:gm_spend_fear(1):spotlight("Rangers")

scene:end_session()

return scene
