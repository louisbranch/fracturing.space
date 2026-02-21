local scene = Scenario.new("environment_prancing_pony_someone_comes_to_town")

-- Capture the arrival of a significant NPC in the tavern.
scene:campaign{
  name = "Environment Prancing Pony Someone Comes to Town",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:npc("Elrond")

-- A new figure arrives with work or a personal connection.
scene:start_session("Someone Comes to Town")
scene:gm_fear(1)

-- Example: introduce a significant NPC as an environment action.
-- NPC hook payload and immediate agenda remain unresolved.
scene:gm_spend_fear(1):spotlight("Elrond")

scene:end_session()

return scene
