local scn = Scenario.new("environment_prancing_pony_someone_comes_to_town")
local dh = scn:system("DAGGERHEART")

-- Capture the arrival of a significant NPC in the tavern.
scn:campaign{
  name = "Environment Prancing Pony Someone Comes to Town",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
scn:npc("Elrond")

-- A new figure arrives with work or a personal connection.
scn:start_session("Someone Comes to Town")
dh:gm_fear(1)

-- Example: introduce a significant NPC as an environment action.
-- NPC hook payload and immediate agenda remain unresolved.
dh:gm_spend_fear(1):spotlight("Elrond")

scn:end_session()

return scn
