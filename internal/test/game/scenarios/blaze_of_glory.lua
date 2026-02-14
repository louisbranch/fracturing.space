local scene = Scenario.new("blaze_of_glory")

-- Frame Ash at death's door to trigger blaze of glory.
scene:campaign{ name = "Blaze of Glory", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Ash", { hp = 0, life_state = "blaze_of_glory" })

-- Start the finale session.
scene:start_session("Finale")

-- Ash triggers the blaze of glory move.
scene:blaze_of_glory("Ash")

-- Close the session on the final blaze.
scene:end_session()
return scene
