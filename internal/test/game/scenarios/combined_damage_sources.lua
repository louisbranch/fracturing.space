local scene = Scenario.new("combined_damage_sources")

-- Introduce two attackers to combine their damage against Bilbo.
scene:campaign{
  name = "Combined Damage Sources",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "damage"
}

scene:pc("Frodo")
scene:pc("Sam")
scene:npc("Bilbo", { hp_max = 6, major_threshold = 3, severe_threshold = 6 })

-- Frodo and Sam land separate hits that combine into one damage total.
-- Missing DSL: apply combined damage directly to an adversary; Bilbo stands in for now.
scene:start_session("Combined Damage")

-- Their damage is summed before comparing against thresholds.
-- Missing DSL: assert the resulting severity tier.
scene:combined_damage{
  target = "Bilbo",
  damage_type = "physical",
  sources = {
    { character = "Frodo", amount = 3 },
    { character = "Sam", amount = 3 }
  }
}

-- Close the session after the combined damage check.
scene:end_session()

return scene
