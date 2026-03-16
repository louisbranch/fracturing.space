local scn = Scenario.new("armor_incoming_reactions")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Armor Incoming Reactions",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo")
scn:pc("Sam")
scn:pc("Merry")
dh:adversary("Orc Raider")

scn:start_session("Armor Incoming Reactions")

dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.runetan-floating-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.runetan-floating-armor",
}

dh:adversary_attack{
  actor = "Orc Raider",
  target = "Frodo",
  difficulty = 12,
  seed = 11,
  armor_reaction = "shifting",
  expect_armor_delta = -1,
  damage_type = "physical"
}

dh:swap_equipment{
  target = "Sam",
  item_id = "armor.dunamis-silkchain",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.dunamis-silkchain",
}

dh:adversary_attack{
  actor = "Orc Raider",
  target = "Sam",
  difficulty = 12,
  seed = 11,
  armor_reaction = "timeslowing",
  armor_reaction_seed = 17,
  expect_armor_delta = -1,
  damage_type = "physical"
}

dh:swap_equipment{
  target = "Merry",
  item_id = "armor.emberwoven-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.emberwoven-armor",
}

dh:adversary_attack{
  actor = "Orc Raider",
  target = "Merry",
  difficulty = 12,
  seed = 11,
  expect_adversary_target = "Orc Raider",
  expect_adversary_stress_delta = 1,
  damage_type = "physical"
}

scn:end_session()

return scn
