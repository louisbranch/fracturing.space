local scene = Scenario.new("loadout_swap")

scene:campaign{ name = "Loadout Swap", system = "DAGGERHEART", gm_mode = "HUMAN" }
scene:pc("Ira", { stress = 1 })
scene:start_session("Loadouts")

scene:swap_loadout{ target = "Ira", card_id = "card:blade", recall_cost = 1, in_rest = false }

scene:end_session()
return scene
