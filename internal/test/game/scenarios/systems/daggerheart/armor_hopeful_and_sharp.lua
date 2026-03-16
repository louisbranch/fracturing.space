local scn = Scenario.new("armor_hopeful_and_sharp")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Armor Hopeful And Sharp",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo")
scn:pc("Pippin")
dh:adversary("Orc Raider")
dh:adversary("Shadow Hound")

scn:start_session("Armor Hopeful And Sharp")

dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.rosewild-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.rosewild-armor",
  expect_armor = 5,
}

dh:action_roll{
  actor = "Frodo",
  trait = "instinct",
  difficulty = 10,
  outcome = "success_hope",
  replace_hope_with_armor = true,
  modifiers = {
    Modifiers.hope("experience"),
  },
  expect_hope_delta = 0,
  expect_armor_delta = -1,
}

dh:swap_equipment{
  target = "Pippin",
  item_id = "armor.spiked-plate-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.spiked-plate-armor",
  expect_armor = 5,
}

dh:attack{
  actor = "Pippin",
  target = "Shadow Hound",
  trait = "instinct",
  difficulty = 0,
  outcome = "success_hope",
  damage_seed = 11,
  attack_range = "melee",
  damage_dice = {
    { sides = 1, count = 1 }
  },
  expect_damage_total = 5,
  damage_type = "physical"
}

scn:end_session()

return scn
