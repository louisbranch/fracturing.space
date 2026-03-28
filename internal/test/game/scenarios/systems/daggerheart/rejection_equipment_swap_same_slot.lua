local scn = Scenario.new("rejection_equipment_swap_same_slot")
local dh = scn:system("DAGGERHEART")

-- Swapping equipment from and to the same slot should be rejected.
scn:campaign{
  name = "Rejection Equipment Swap Same Slot",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rejection"
}

scn:pc("Frodo")

scn:start_session("Rejection")

-- Attempt to swap from "active" to "active" — no-op.
dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.elundrian-chain-armor",
  item_type = "armor",
  from = "active",
  to = "active",
  expect_error = {code = "INTERNAL"}
}

scn:end_session()
return scn
