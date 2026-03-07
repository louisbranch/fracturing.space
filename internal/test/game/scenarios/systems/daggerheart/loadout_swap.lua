local scn = Scenario.new("loadout_swap")
local dh = scn:system("DAGGERHEART")

-- Introduce Gandalf so a mid-session loadout swap matters.
scn:campaign{ name = "Loadout Swap", system = "DAGGERHEART", gm_mode = "HUMAN" }
scn:pc("Gandalf", { stress = 1 })

-- Open a session to test a mid-scene loadout swap.
scn:start_session("Loadouts")

-- Gandalf recalls a blade as part of a mid-session swap.
dh:swap_loadout{ target = "Gandalf", card_id = "card:blade", recall_cost = 1, in_rest = false }

-- Close the session once the swap lands.
scn:end_session()
return scn
