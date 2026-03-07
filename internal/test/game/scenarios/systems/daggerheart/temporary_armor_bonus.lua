local scene = Scenario.new("temporary_armor_bonus")
local dh = scene:system("DAGGERHEART")

scene:campaign{
  name = "Temporary Armor Bonus",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "armor"
}

scene:pc("Gandalf", { armor = 3 })

scene:start_session("Armor Bonus")

dh:temporary_armor{
  target = "Gandalf",
  source = "ritual",
  duration = "short_rest",
  amount = 2,
  source_id = "blessing:1",
  expect_target = "Gandalf",
  expect_armor_delta = 2,
}

dh:rest{
  type = "short",
  party_size = 1,
  expect_target = "Gandalf",
  expect_armor_delta = -2,
}

dh:temporary_armor{
  target = "Gandalf",
  source = "warding",
  duration = "long_rest",
  amount = 2,
  source_id = "long:1",
  expect_target = "Gandalf",
  expect_armor_delta = 2,
}

dh:rest{
  type = "long",
  party_size = 1,
  expect_target = "Gandalf",
  expect_armor_delta = -2,
}

scene:end_session()

return scene
