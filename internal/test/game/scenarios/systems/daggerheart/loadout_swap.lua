local scene = Scenario.new("loadout_swap")
local dh = scene:system("DAGGERHEART")

-- Introduce Gandalf so a mid-session loadout swap matters.
scene:campaign{ name = "Loadout Swap", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Gandalf", { stress = 1 })

-- Open a session to test a mid-scene loadout swap.
scene:start_session("Loadouts")

-- Gandalf recalls a blade as part of a mid-session swap.
dh:swap_loadout{ target = "Gandalf", card_id = "card:blade", recall_cost = 1, in_rest = false }

-- Close the session once the swap lands.
scene:end_session()
return scene
