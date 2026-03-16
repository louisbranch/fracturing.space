local scn = Scenario.new("subclass_progression_tracks")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Subclass Progression Tracks",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "progression"
}

scn:pc("Frodo")

scn:start_session("Subclass Progression")

dh:level_up{
  target = "Frodo",
  level_after = 2,
  advancements = {
    { type = "upgraded_subclass" },
    { type = "add_hp_slots" },
  },
  expect_level = 2,
  expect_subclass_track_count = 1,
  expect_primary_subclass_rank = "specialization",
  expect_active_feature_ids = {
    "feature.stalwart-unwavering",
    "feature.stalwart-unrelenting",
  },
}

dh:level_up{
  target = "Frodo",
  level_after = 3,
  advancements = {
    { type = "upgraded_subclass" },
    { type = "add_stress_slots" },
  },
  expect_level = 3,
  expect_subclass_track_count = 1,
  expect_primary_subclass_rank = "mastery",
  expect_active_feature_ids = {
    "feature.stalwart-unwavering",
    "feature.stalwart-unrelenting",
    "feature.stalwart-undaunted",
  },
}

scn:end_session()

return scn
