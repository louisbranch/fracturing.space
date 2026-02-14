local scene = Scenario.new("reaction_flow")

-- Introduce Wren to highlight reaction timing.
scene:campaign{ name = "Reaction Flow", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Wren")

-- Open a session to test reaction timing.
scene:start_session("Reaction")

-- Wren makes a reaction roll under pressure.
scene:reaction{ actor = "Wren", trait = "agility", difficulty = 8, outcome = "hope" }

-- Close the session after the reaction.
scene:end_session()
return scene
