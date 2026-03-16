local scn = Scenario.new("armor_feature_rules")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Armor Feature Rules",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo")
scn:pc("Sam")
scn:pc("Merry")
scn:pc("Pippin")

scn:start_session("Armor Rules")

dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.elundrian-chain-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.elundrian-chain-armor",
  expect_armor = 4,
}

dh:combined_damage{
  target = "Frodo",
  damage_type = "magic",
  sources = {
    { amount = 7, character = "gm" }
  },
  expect_hp_delta = -1,
  expect_armor_delta = 0,
  expect_stress_delta = 0,
}

dh:swap_equipment{
  target = "Sam",
  item_id = "armor.full-fortified-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.full-fortified-armor",
  expect_armor = 4,
}

dh:combined_damage{
  target = "Sam",
  damage_type = "physical",
  sources = {
    { amount = 10, character = "gm" }
  },
  expect_hp_delta = -1,
  expect_armor_delta = -1,
  expect_stress_delta = 0,
}

dh:swap_equipment{
  target = "Merry",
  item_id = "armor.runes-of-fortification",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.runes-of-fortification",
  expect_armor = 6,
}

dh:combined_damage{
  target = "Merry",
  damage_type = "physical",
  sources = {
    { amount = 7, character = "gm" }
  },
  expect_hp_delta = -1,
  expect_armor_delta = -1,
  expect_stress_delta = 1,
}

dh:swap_equipment{
  target = "Pippin",
  item_id = "armor.irontree-breastplate-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.irontree-breastplate-armor",
  expect_armor = 4,
}

dh:combined_damage{ target = "Pippin", damage_type = "physical", sources = { { amount = 1, character = "gm" } }, expect_hp_delta = 0, expect_armor_delta = -1, expect_stress_delta = 0 }
dh:combined_damage{ target = "Pippin", damage_type = "physical", sources = { { amount = 1, character = "gm" } }, expect_hp_delta = 0, expect_armor_delta = -1, expect_stress_delta = 0 }
dh:combined_damage{ target = "Pippin", damage_type = "physical", sources = { { amount = 1, character = "gm" } }, expect_hp_delta = 0, expect_armor_delta = -1, expect_stress_delta = 0 }
dh:combined_damage{ target = "Pippin", damage_type = "physical", sources = { { amount = 1, character = "gm" } }, expect_hp_delta = 0, expect_armor_delta = -1, expect_stress_delta = 0 }

dh:combined_damage{
  target = "Pippin",
  damage_type = "physical",
  sources = {
    { amount = 21, character = "gm" }
  },
  expect_hp_delta = -2,
  expect_armor_delta = 0,
  expect_stress_delta = 0,
}

scn:end_session()

return scn
