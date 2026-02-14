local scene = Scenario.new("damage_thresholds_example")

-- Recreate the guardian damage threshold example.
scene:campaign{
  name = "Damage Thresholds Example",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scene:npc("Galadriel", { hp_max = 12, hp = 12, major_threshold = 8, severe_threshold = 16 })

-- A sequence of hits crosses Minor, Major, Severe, and Massive tiers.
scene:start_session("Thresholds")

-- Example: 8+ is Major, 16+ is Severe, 32+ is Massive.
-- Missing DSL: assert tier mapping and HP marked for each tier.
scene:combined_damage{
  target = "Galadriel",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 8 }
  }
}
scene:combined_damage{
  target = "Galadriel",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 16 }
  }
}
scene:combined_damage{
  target = "Galadriel",
  damage_type = "physical",
  sources = {
    { character = "GM", amount = 32 }
  }
}

-- Close the session after the threshold checks.
scene:end_session()

return scene
