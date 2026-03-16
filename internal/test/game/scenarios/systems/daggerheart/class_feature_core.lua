local scn = Scenario.new("class_feature_core")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Class Feature Core",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "class features"
}

scn:pc("Aegis", { hope = 5, armor = 0, armor_max = 3 })
scn:pc("Rune", { skip_system_readiness = true })

dh:creation_workflow{
  target = "Rune",
  class_id = "class.wizard",
  subclass_id = "subclass.school-of-war",
  domain_card_ids = {
    "domain_card.codex-pattern-study",
    "domain_card.codex-pattern-study"
  },
  heritage = {
    first_feature_ancestry_id = "heritage.human",
    second_feature_ancestry_id = "heritage.human",
    community_id = "heritage.highborne"
  },
  expect_class_id = "class.wizard",
  expect_subclass_id = "subclass.school-of-war"
}

scn:start_session("Class Feature Core")

dh:class_feature{
  target = "Aegis",
  feature = "frontline_tank",
  expect_hope_delta = -3,
  expect_armor_delta = 2
}

dh:class_feature{
  target = "Rune",
  feature = "strange_patterns_choice",
  number = 7,
  expect_strange_patterns_number = 7
}

scn:end_session()

return scn
