local scn = Scenario.new("environment_dark_tower_usurpation_beginning_of_end")
local dh = scn:system("DAGGERHEART")

-- Capture the divine siege countdown and fear gain on completion.
scn:campaign{
  name = "Environment Dark Tower Usurpation Beginning of End",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Saruman")

-- The ritual escalates into a siege of the gods.
scn:start_session("Beginning of the End")

-- Major-damage tick branches remain unresolved in this fixture.
dh:scene_countdown_create{ name = "Divine Siege", kind = "consequence", current = 0, max = 10, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 14, outcome = "fear" }
dh:scene_countdown_update{ name = "Divine Siege", delta = 1, reason = "fear_outcome" }

scn:end_session()

return scn
