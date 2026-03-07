local scene = Scenario.new("blaze_of_glory")
local dh = scene:system("DAGGERHEART")

-- Frame Frodo at death's door to trigger blaze of glory.
scene:campaign{ name = "Blaze of Glory", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Frodo", { hp = 0, life_state = "blaze_of_glory" })

-- Start the finale session.
scene:start_session("Finale")

-- Frodo triggers the blaze of glory move.
dh:blaze_of_glory("Frodo")

-- Close the session on the final blaze.
scene:end_session()
return scene
