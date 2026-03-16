local scn = Scenario.new("subclass_multiclass_tracks")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Subclass Multiclass Tracks",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "progression"
}

scn:pc("Frodo")

scn:start_session("Subclass Multiclass")

dh:level_up{ target = "Frodo", level_after = 2, advancements = {
  { type = "trait_increase", trait = "agility" },
  { type = "add_hp_slots" },
} }
dh:level_up{ target = "Frodo", level_after = 3, advancements = {
  { type = "trait_increase", trait = "strength" },
  { type = "add_hp_slots" },
} }
dh:level_up{ target = "Frodo", level_after = 4, advancements = {
  { type = "trait_increase", trait = "finesse" },
  { type = "add_hp_slots" },
} }
dh:level_up{ target = "Frodo", level_after = 5, advancements = {
  { type = "increase_proficiency" },
} }

dh:level_up{
  target = "Frodo",
  level_after = 6,
  advancements = {
    { type = "multiclass", multiclass = {
      secondary_class_id = "class.bard",
      secondary_subclass_id = "subclass.wordsmith",
      spellcast_trait = "presence",
      domain_id = "domain.codex",
    }},
  },
  expect_level = 6,
  expect_subclass_track_count = 2,
  expect_primary_subclass_rank = "foundation",
  expect_multiclass_subclass_id = "subclass.wordsmith",
  expect_active_feature_ids = {
    "feature.stalwart-unwavering",
    "feature.wordsmith-foundation",
  },
}

scn:end_session()

return scn
