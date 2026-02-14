local scene = Scenario.new("gm_move_severity")

-- Stage a tense clash to show soft vs hard GM moves.
scene:campaign{
  name = "GM Move Severity",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "pressure"
}

scene:pc("Frodo")
scene:adversary("Nazgul")

-- The GM sets up a pressure cooker with fear on hand.
scene:start_session("Severity")
scene:gm_fear(3)

-- Frodo rolls with Fear, handing control back to the GM.
-- Missing DSL: assert fear increases by 1.
scene:attack{ actor = "Frodo", target = "Nazgul", trait = "instinct", difficulty = 0, outcome = "fear" }

-- A small fear spend suggests a softer GM move.
scene:gm_spend_fear(1):spotlight("Nazgul")
-- A larger fear spend hints at a harder, more costly move.
-- Missing DSL: annotate soft vs hard move consequences.
scene:gm_spend_fear(2):spotlight("Nazgul")

-- Close the session after the GM move cadence.
scene:end_session()

return scene
