local scene = Scenario.new("ranged_arcane_artillery")

-- Capture the Saruman's arcane artillery fear action.
scene:campaign{
  name = "Ranged Arcane Artillery",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
scene:pc("Sam")
scene:adversary("Saruman")

-- The wizard spends Fear to blast all targets with a reaction roll.
scene:start_session("Arcane Artillery")
scene:gm_fear(1)

-- Example: all targets roll Agility or take 2d12 magic damage (half on success).
scene:gm_spend_fear(1):spotlight("Saruman")
scene:group_reaction{
  targets = {"Frodo", "Sam"},
  trait = "agility",
  difficulty = 15,
  damage = 12,
  damage_type = "magic",
  half_damage_on_success = true,
  source = "arcane_artillery"
}

scene:end_session()

return scene
