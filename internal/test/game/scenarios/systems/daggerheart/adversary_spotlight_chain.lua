local scene = Scenario.new("adversary_spotlight_chain")
local dh = scene:system("DAGGERHEART")

-- Gather threats so the spotlight can bounce between them.
scene:campaign{
  name = "Adversary Spotlight Chain",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spotlight"
}

scene:pc("Frodo")
dh:adversary("Nazgul")
dh:adversary("Golum")

-- The GM bounces the spotlight between threats in rapid succession.
scene:start_session("Spotlight Chain")
dh:gm_fear(3)

-- Each fear spend yanks the spotlight to a different adversary.
dh:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
dh:gm_spend_fear(1):spotlight("Golum", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
dh:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })

-- Close the session after the spotlight volley.
scene:end_session()

return scene
