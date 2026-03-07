local scn = Scenario.new("damage_thresholds_example")
local dh = scn:system("DAGGERHEART")

-- Recreate the guardian damage threshold example.
scn:campaign{
  name = "Damage Thresholds Example",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scn:npc("Galadriel", { hp_max = 12, hp = 12, major_threshold = 8, severe_threshold = 16 })

-- A sequence of hits crosses Minor, Major, Severe, and Massive tiers.
scn:start_session("Thresholds")

-- Example: 8+ is Major, 16+ is Severe, 32+ is Massive.
-- Missing DSL: assert tier mapping and HP marked for each tier.
dh:combined_damage{
  target = "Galadriel",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 8 }
  }
}
dh:combined_damage{
  target = "Galadriel",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 16 }
  }
}
dh:combined_damage{
  target = "Galadriel",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 32 }
  }
}

-- Close the session after the threshold checks.
scn:end_session()

return scn
