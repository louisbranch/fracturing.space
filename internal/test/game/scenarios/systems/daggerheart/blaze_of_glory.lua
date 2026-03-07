local scn = Scenario.new("blaze_of_glory")
local dh = scn:system("DAGGERHEART")

-- Frame Frodo at death's door to trigger blaze of glory.
scn:campaign{ name = "Blaze of Glory", system = "DAGGERHEART", gm_mode = "HUMAN" }
scn:pc("Frodo", { hp = 0, life_state = "blaze_of_glory" })

-- Start the finale session.
scn:start_session("Finale")

-- Frodo triggers the blaze of glory move.
dh:blaze_of_glory("Frodo")

-- Close the session on the final blaze.
scn:end_session()
return scn
