local scn = Scenario.new("equipment_swap")
local dh = scn:system("DAGGERHEART")

-- Verify equipment swaps between active and inventory slots.
scn:campaign{
  name = "Equipment Swap",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "inventory"
}

scn:pc("Frodo")

-- Equip a weapon from inventory to active.
scn:start_session("Equip Swap")
dh:swap_equipment{
  target = "Frodo",
  item_id = "weapon.longsword",
  item_type = "weapon",
  from = "active",
  to = "inventory",
}

-- Swap weapon: move longsword back to active.
dh:swap_equipment{
  target = "Frodo",
  item_id = "weapon.longsword",
  item_type = "weapon",
  from = "inventory",
  to = "active",
}

-- Swap armor from inventory to active.
dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.leather-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
}

scn:end_session()

return scn
