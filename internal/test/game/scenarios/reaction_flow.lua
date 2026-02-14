local scene = Scenario.new("reaction_flow")

scene:campaign{ name = "Reaction Flow", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Wren")
scene:start_session("Reaction")

scene:reaction{ actor = "Wren", trait = "agility", difficulty = 8, outcome = "hope" }

scene:end_session()
return scene
