local scn = Scenario.new("armor_quiet")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Armor Quiet",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo")

scn:start_session("Armor Quiet")

dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.tyris-soft-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.tyris-soft-armor",
  expect_armor = 5,
}

dh:action_roll{
  actor = "Frodo",
  trait = "agility",
  difficulty = 14,
  outcome = "success_hope",
  context = "move_silently",
}

scn:end_session()

return scn
