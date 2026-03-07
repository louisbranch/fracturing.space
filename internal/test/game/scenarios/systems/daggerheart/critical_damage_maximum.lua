local scene = Scenario.new("critical_damage_maximum")
local dh = scene:system("DAGGERHEART")

-- Capture the critical success example with max damage dice.
scene:campaign{
  name = "Critical Damage Maximum",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "crit"
}

scene:pc("Gandalf")
dh:adversary("Nazgul")

-- Gandalf critically succeeds on an attack and doubles up damage logic.
scene:start_session("Critical Damage")

-- Example: critical success, start with max 2d6 = 12, then roll 2d6 (4, 5) +1.
-- Missing DSL: apply max-dice bonus before rolling damage.
dh:attack{
  actor = "Gandalf",
  target = "Nazgul",
  trait = "spellcast",
  difficulty = 0,
  outcome = "critical",
  damage_type = "magic"
}
dh:damage_roll{
  actor = "Gandalf",
  damage_dice = { { sides = 6, count = 2 } },
  modifier = 1
}

-- Close the session after the critical damage example.
scene:end_session()

return scene
