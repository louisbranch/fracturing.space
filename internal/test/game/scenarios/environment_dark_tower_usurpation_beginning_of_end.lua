local scene = Scenario.new("environment_dark_tower_usurpation_beginning_of_end")

-- Capture the divine siege countdown and fear gain on completion.
scene:campaign{
  name = "Environment Dark Tower Usurpation Beginning of End",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Saruman")

-- The ritual escalates into a siege of the gods.
scene:start_session("Beginning of the End")

-- Major-damage tick branches remain unresolved in this fixture.
scene:countdown_create{ name = "Divine Siege", kind = "consequence", current = 0, max = 10, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 14, outcome = "fear" }
scene:countdown_update{ name = "Divine Siege", delta = 1, reason = "fear_outcome" }

scene:end_session()

return scene
