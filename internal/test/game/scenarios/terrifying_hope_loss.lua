local scene = Scenario.new("terrifying_hope_loss")

-- Capture the Terrifying feature forcing Hope loss in Close range.
scene:campaign{
  name = "Terrifying Hope Loss",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "fear"
}

scene:pc("Sam", { armor = 1 })
scene:pc("Frodo")
scene:pc("Gandalf")
scene:pc("Aragorn")
scene:adversary("Nazgul")

-- The Nazgul lands a heavy blow and its Terrifying feature triggers.
scene:start_session("Terrifying Strike")

-- Example: 10 damage is Major; Sam marks armor to reduce to Minor.
-- Partial mapping: explicit post-trigger GM Fear gain is represented.
-- Missing DSL: grouped Hope-loss fanout with per-character lower-bound enforcement.
scene:adversary_attack{
  actor = "Nazgul",
  target = "Sam",
  difficulty = 0,
  damage_type = "physical"
}
scene:gm_fear(1)

-- Close the session after the terrifying strike.
scene:end_session()

return scene
