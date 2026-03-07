local scene = Scenario.new("minion_overflow_damage")
local dh = scene:system("DAGGERHEART")

-- Showcase Minion (X) overflow defeating extra minions.
scene:campaign{
  name = "Minion Overflow Damage",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "minions"
}

scene:pc("Frodo")
dh:adversary("Moria Rat A")
dh:adversary("Moria Rat B")
dh:adversary("Moria Rat C")

-- One hit drops multiple rats when damage meets Minion (3).
scene:start_session("Minion Overflow")

-- Example: 6 damage defeats the target plus two more Minions.
-- Missing DSL: apply Minion (3) overflow and select extra targets.
dh:combined_damage{
  target = "Moria Rat A",
  damage_type = "physical",
  sources = {
    { character = "Frodo", amount = 6 }
  }
}

scene:end_session()

return scene
