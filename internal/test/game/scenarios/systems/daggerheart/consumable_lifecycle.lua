local scn = Scenario.new("consumable_lifecycle")
local dh = scn:system("DAGGERHEART")

-- Verify consumable acquire and use lifecycle with quantity tracking.
scn:campaign{
  name = "Consumable Lifecycle",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "inventory"
}

scn:pc("Frodo")

-- Acquire a healing potion.
scn:start_session("Consumable")
dh:acquire_consumable{
  target = "Frodo",
  consumable_id = "healing_potion",
  quantity_before = 0,
  quantity_after = 1,
}

-- Acquire a second healing potion (stack).
dh:acquire_consumable{
  target = "Frodo",
  consumable_id = "healing_potion",
  quantity_before = 1,
  quantity_after = 2,
}

-- Use one healing potion.
dh:use_consumable{
  target = "Frodo",
  consumable_id = "healing_potion",
  quantity_before = 2,
  quantity_after = 1,
}

-- Use last healing potion.
dh:use_consumable{
  target = "Frodo",
  consumable_id = "healing_potion",
  quantity_before = 1,
  quantity_after = 0,
}

scn:end_session()

return scn
