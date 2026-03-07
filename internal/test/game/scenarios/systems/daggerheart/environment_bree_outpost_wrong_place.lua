local scene = Scenario.new("environment_bree_outpost_wrong_place")
local dh = scene:system("DAGGERHEART")

-- Capture the ambush by thieves in a dark alley.
scene:campaign{
  name = "Environment Bree Outpost Wrong Place",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Orc Captain")
dh:adversary("Orc Lackeys")
dh:adversary("Orc Lieutenant")

-- Thieves emerge at close range when the party is isolated.
scene:start_session("Wrong Place, Wrong Time")
dh:gm_fear(1)

-- Example: spend Fear to introduce a robber group at Close range.
dh:adversary("Orc Lackey Reinforcement 1")
dh:adversary("Orc Lackey Reinforcement 2")
-- Party-size scaling and exact range placement remain unresolved.
dh:gm_spend_fear(1):spotlight("Orc Captain")

scene:end_session()

return scene
