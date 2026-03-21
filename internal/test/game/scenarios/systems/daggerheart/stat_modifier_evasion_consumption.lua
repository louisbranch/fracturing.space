local scn = Scenario.new("stat_modifier_evasion_consumption")
local dh = scn:system("DAGGERHEART")

-- Verify that evasion stat modifiers can be applied and clear on short rest.
scn:campaign{
  name = "Stat Modifier Evasion Consumption",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "stat_modifiers"
}

scn:pc("Sentinel", { evasion = 3, armor = 0 })

scn:start_session("Evasion Modifiers")
dh:gm_fear(2)

-- Apply an evasion modifier with a short-rest clear trigger.
dh:apply_stat_modifier{
  target = "Sentinel",
  add = {
    { id = "mod-evasion-wall", target = "evasion", delta = 100, label = "Wall of Iron", source = "domain_card", clear_triggers = { "SHORT_REST" } }
  },
  source = "domain_card.wall_of_iron",
  expect_active_count = 1,
  expect_added_count = 1
}

-- Short rest clears the modifier; evasion returns to 3.
dh:rest{
  type = "short",
  participants = { "Sentinel" }
}

-- Re-applying the same modifier should succeed because the rest cleared it.
dh:apply_stat_modifier{
  target = "Sentinel",
  add = {
    { id = "mod-evasion-wall", target = "evasion", delta = 100, label = "Wall of Iron", source = "domain_card", clear_triggers = { "SHORT_REST" } }
  },
  source = "domain_card.wall_of_iron",
  expect_active_count = 1,
  expect_added_count = 1
}

scn:end_session()

return scn
