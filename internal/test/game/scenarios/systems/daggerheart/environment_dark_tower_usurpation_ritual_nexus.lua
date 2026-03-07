local scene = Scenario.new("environment_dark_tower_usurpation_ritual_nexus")
local dh = scene:system("DAGGERHEART")

-- Capture the ritual backlash on failures with Fear.
scene:campaign{
  name = "Environment Dark Tower Usurpation Ritual Nexus",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Saruman")

-- A Fear failure triggers magical backlash.
scene:start_session("Ritual Nexus")

-- Stress roll (1d4) on failure with Fear remains unresolved.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 20, outcome = "fear" }
dh:apply_roll_outcome{}

scene:end_session()

return scene
