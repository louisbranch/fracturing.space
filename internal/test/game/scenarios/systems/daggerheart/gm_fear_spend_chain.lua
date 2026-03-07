local scene = Scenario.new("gm_fear_spend_chain")
local dh = scene:system("DAGGERHEART")

-- Establish a scene for rapid fear spending.
scene:campaign{
  name = "GM Fear Spend Chain",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "gm_fear"
}

scene:pc("Frodo")
dh:adversary("Nazgul")

-- The GM starts with fear and spends it in quick succession.
scene:start_session("GM Fear")
dh:gm_fear(5)

-- Two spotlight spends show how fear accelerates the GM's cadence.
dh:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
dh:gm_spend_fear(2):spotlight("Nazgul", { expect_gm_fear_delta = -2, expect_gm_move = "spotlight", expect_gm_fear_spent = 2 })

-- Close the session after the fear spend chain.
scene:end_session()

return scene
