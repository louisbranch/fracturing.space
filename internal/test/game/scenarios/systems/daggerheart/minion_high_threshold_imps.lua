local scn = Scenario.new("minion_high_threshold_imps")
local dh = scn:system("DAGGERHEART")

-- Reflect the Idolizing Imp Minion (8) overflow threshold.
scn:campaign{
  name = "Minion High Threshold Imps",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "minions"
}

scn:pc("Frodo")
dh:adversary("Goblin A")
dh:adversary("Goblin B")

-- Heavier hits are needed to drop extra imps at once.
scn:start_session("Imp Overflow")

-- Example: 8 damage defeats the target plus one more Minion.
-- Missing DSL: apply Minion (8) overflow.
dh:combined_damage{
  target = "Goblin A",
  damage_type = "magic",
  sources = {
    { character = "Frodo", amount = 8 }
  }
}

scn:end_session()

return scn
