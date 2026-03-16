local scn = Scenario.new("armor_swap_effective_stats")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Armor Swap Effective Stats",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "equipment"
}

scn:pc("Frodo")

scn:start_session("Armor Swap")

dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.chainmail-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.chainmail-armor",
  expect_evasion = 8,
  expect_major_threshold = 7,
  expect_severe_threshold = 15,
  expect_armor_max = 4,
  expect_armor = 4,
}

dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.channeling-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.channeling-armor",
  expect_evasion = 9,
  expect_major_threshold = 13,
  expect_severe_threshold = 36,
  expect_armor_max = 5,
  expect_armor = 5,
  expect_spellcast_roll_bonus = 1,
}

scn:end_session()

return scn
