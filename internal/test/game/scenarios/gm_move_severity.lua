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
scene:attack{ actor = "Frodo", target = "Nazgul", trait = "instinct", difficulty = 0, outcome = "fear", expect_gm_fear_delta = 1, expect_spotlight = "gm", expect_requires_complication = true }

-- A small fear spend suggests a softer GM move.
scene:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1, expect_gm_move_description = "spotlight Nazgul", expect_gm_move_severity = "soft" })
-- A larger fear spend hints at a harder, more costly move.
scene:gm_spend_fear(2):spotlight("Nazgul", { expect_gm_fear_delta = -2, expect_gm_move = "spotlight", expect_gm_fear_spent = 2, expect_gm_move_description = "spotlight Nazgul", expect_gm_move_severity = "hard" })

-- Close the session after the GM move cadence.
scene:end_session()

return scene
