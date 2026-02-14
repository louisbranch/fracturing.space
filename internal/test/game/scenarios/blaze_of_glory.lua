local scene = Scenario.new("blaze_of_glory")

scene:campaign{ name = "Blaze of Glory", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Ash", { hp = 0, life_state = "blaze_of_glory" })
scene:start_session("Finale")

scene:blaze_of_glory("Ash")

scene:end_session()
return scene
