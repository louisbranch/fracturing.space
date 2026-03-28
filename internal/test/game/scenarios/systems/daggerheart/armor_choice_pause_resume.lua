local scn = Scenario.new("armor_choice_pause_resume")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Armor Choice Pause Resume",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scn:pc("Frodo")
scn:pc("Sam")
dh:adversary("Orc Raider")

scn:start_session("Armor Choice Pause Resume")

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
  difficulty = 1,
  seed = 11,
  damage_type = "physical",
  require_defense_choice = true,
  armor_reaction = "shifting",
  expect_choice_stage = "incoming_attack_defense",
  expect_choice_options = { "armor.shifting", "armor.decline" },
  expect_armor_delta = -1,
}

dh:combined_damage{
  target = "Sam",
  damage_type = "physical",
  require_mitigation_choice = true,
  base_armor_decision = "decline",
  expect_choice_stage = "damage_mitigation",
  expect_choice_options = { "armor.base_slot", "armor.decline" },
  sources = {
    { amount = 4, character = "gm" }
  },
  expect_hp_delta = -2,
  expect_armor_delta = 0,
}

scn:end_session()

return scn
