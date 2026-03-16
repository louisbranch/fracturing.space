local scn = Scenario.new("armor_last_chance_reactions")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Armor Last Chance Reactions",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo")
scn:start_session("Armor Last Chance Reactions")

dh:swap_equipment{
  target = "Frodo",
  item_id = "armor.harrowbone-armor",
  item_type = "armor",
  from = "inventory",
  to = "active",
  expect_equipped_armor_id = "armor.harrowbone-armor",
  expect_armor = 4,
}

dh:combined_damage{ target = "Frodo", damage_type = "physical", sources = { { amount = 1, character = "gm" } }, expect_armor_delta = -1 }
dh:combined_damage{ target = "Frodo", damage_type = "physical", sources = { { amount = 1, character = "gm" } }, expect_armor_delta = -1 }
dh:combined_damage{ target = "Frodo", damage_type = "physical", sources = { { amount = 1, character = "gm" } }, expect_armor_delta = -1 }

dh:combined_damage{
  target = "Frodo",
  damage_type = "physical",
  armor_reaction = "resilient",
  armor_reaction_seed = 1,
  sources = {
    { amount = 9, character = "gm" }
  },
  expect_armor_delta = 0
}

scn:end_session()

return scn
