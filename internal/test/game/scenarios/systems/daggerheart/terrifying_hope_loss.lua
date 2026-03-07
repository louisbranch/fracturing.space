local scn = Scenario.new("terrifying_hope_loss")
local dh = scn:system("DAGGERHEART")

-- Capture the Terrifying feature forcing Hope loss in Close range.
scn:campaign{
  name = "Terrifying Hope Loss",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "fear"
}

scn:pc("Sam", { armor = 1 })
scn:pc("Frodo")
scn:pc("Gandalf")
scn:pc("Aragorn")
dh:adversary("Nazgul")

-- The Nazgul lands a heavy blow and its Terrifying feature triggers.
scn:start_session("Terrifying Strike")

-- Example: 10 damage is Major; Sam marks armor to reduce to Minor.
-- Partial mapping: explicit post-trigger GM Fear gain is represented.
-- Missing DSL: grouped Hope-loss fanout with per-character lower-bound enforcement.
dh:adversary_attack{
  actor = "Nazgul",
  target = "Sam",
  difficulty = 0,
  damage_type = "physical"
}
dh:gm_fear(1)

-- Close the session after the terrifying strike.
scn:end_session()

return scn
