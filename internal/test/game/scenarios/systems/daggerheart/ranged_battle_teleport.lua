local scn = Scenario.new("ranged_battle_teleport")
local dh = scn:system("DAGGERHEART")

-- Capture the Saruman's battle teleport stress spend.
scn:campaign{
  name = "Ranged Battle Teleport",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary"
}

scn:pc("Frodo")
dh:adversary("Saruman")

-- The wizard teleports before or after a standard attack.
scn:start_session("Battle Teleport")

-- Example: mark Stress to teleport within Far range.
dh:adversary_attack{
  actor = "Saruman",
  target = "Frodo",
  difficulty = 0,
  teleport_range = "far",
  teleport_stress_cost = 1,
  damage_type = "magic"
}

scn:end_session()

return scn
