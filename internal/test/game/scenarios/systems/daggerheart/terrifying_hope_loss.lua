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

-- The hunter lands a heavy blow and its Terrifying feature triggers.
scn:start_session("Terrifying Strike")

dh:adversary_attack{
  actor = "Nazgul",
  target = "Sam",
  feature_id = "feature.nazgul-terrifying",
  difficulty = 0,
  damage_type = "physical"
}
dh:expect_gm_fear(5)

-- Close the session after the terrifying strike.
scn:end_session()

return scn
