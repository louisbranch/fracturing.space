local scn = Scenario.new("creation_mixed_heritage_beastbound")
local dh = scn:system("DAGGERHEART")

-- Set up an explicit creation-flow scenario instead of relying on readiness defaults.
scn:campaign{
  name = "Creation Mixed Heritage Beastbound",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "creation"
}

-- Add one unready PC so the scenario can drive creation explicitly.
scn:pc("Mira", { skip_system_readiness = true })

-- Beastbound should reject creation without the required companion setup.
dh:creation_workflow{
  target = "Mira",
  class_id = "class.ranger",
  subclass_id = "subclass.beastbound",
  heritage = {
    first_feature_ancestry_id = "heritage.dwarf",
    second_feature_ancestry_id = "heritage.elf",
    ancestry_label = "Stoneleaf",
    community_id = "heritage.highborne"
  },
  expect_error = {
    code = "INVALID_ARGUMENT",
    contains = "companion"
  }
}

-- Mixed heritage plus Beastbound companion setup should persist on the sheet.
dh:creation_workflow{
  target = "Mira",
  class_id = "class.ranger",
  subclass_id = "subclass.beastbound",
  heritage = {
    first_feature_ancestry_id = "heritage.dwarf",
    second_feature_ancestry_id = "heritage.elf",
    ancestry_label = "Stoneleaf",
    community_id = "heritage.highborne"
  },
  companion = {
    animal_kind = "Raccoon",
    name = "Rocket",
    experience_ids = { "companion-experience.navigation", "companion-experience.scout" },
    attack_description = "Short range concussion blast",
    damage_type = "physical"
  },
  expect_class_id = "class.ranger",
  expect_subclass_id = "subclass.beastbound",
  expect_heritage_label = "Stoneleaf",
  expect_first_feature_ancestry_id = "heritage.dwarf",
  expect_second_feature_ancestry_id = "heritage.elf",
  expect_community_id = "heritage.highborne",
  expect_companion_present = true,
  expect_companion_name = "Rocket",
  expect_companion_animal_kind = "Raccoon",
  expect_companion_damage_type = "physical"
}

-- Starting the session after explicit creation should seed fear from the one PC.
scn:start_session("Creation")
dh:expect_gm_fear(1)
scn:end_session()

return scn
