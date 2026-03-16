local scn = Scenario.new("multiclass_unlock")
local dh = scn:system("DAGGERHEART")

-- Verify multiclass advancement at level 5+.
scn:campaign{
  name = "Multiclass Unlock",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "progression"
}

scn:pc("Frodo")

-- Advance to level 5 then take multiclass advancement at level 6.
scn:start_session("Multiclass")

-- Levels 1 through 5.
dh:level_up{ target = "Frodo", level_after = 2, advancements = { { type = "trait_increase", trait = "agility" }, { type = "add_hp_slots" } } }
dh:level_up{ target = "Frodo", level_after = 3, advancements = { { type = "trait_increase", trait = "strength" }, { type = "add_hp_slots" } } }
dh:level_up{ target = "Frodo", level_after = 4, advancements = { { type = "trait_increase", trait = "finesse" }, { type = "add_hp_slots" } } }
dh:level_up{ target = "Frodo", level_after = 5, advancements = { { type = "increase_proficiency" } } }

-- Level 5 to 6: take multiclass advancement.
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
}

scn:end_session()

return scn
