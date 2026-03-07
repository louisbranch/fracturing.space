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
  item_id = "shortsword_01",
  item_type = "weapon",
  from = "inventory",
  to = "active",
}

-- Swap weapon: move shortsword back to inventory.
dh:swap_equipment{
  target = "Frodo",
  item_id = "shortsword_01",
  item_type = "weapon",
  from = "active",
  to = "inventory",
}

-- Equip armor from none to active.
dh:swap_equipment{
  target = "Frodo",
  item_id = "leather_armor_01",
  item_type = "armor",
  from = "none",
  to = "active",
}

scn:end_session()

return scn
