local scene = Scenario.new("ranged_battle_teleport")

-- Capture the Saruman's battle teleport stress spend.
scene:campaign{
  name = "Ranged Battle Teleport",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scene:pc("Frodo")
scene:adversary("Saruman")

-- The wizard teleports before or after a standard attack.
scene:start_session("Battle Teleport")

-- Example: mark Stress to teleport within Far range.
scene:adversary_attack{
  actor = "Saruman",
  target = "Frodo",
  difficulty = 0,
  teleport_range = "far",
  teleport_stress_cost = 1,
  damage_type = "magic"
}

scene:end_session()

return scene
