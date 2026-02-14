local scene = Scenario.new("fear_floor")

-- Set a simple scene to drain the fear pool to zero.
scene:campaign{
  name = "Fear Floor",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "fear"
}

scene:pc("Frodo")
scene:adversary("Nazgul")

-- The GM spends fear until the pool runs dry.
scene:start_session("Fear Floor")
scene:gm_fear(2)

-- Two spends should bring fear to zero without going negative.
scene:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
scene:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
scene:gm_fear(0)

-- Close the session once fear hits the floor.
scene:end_session()

return scene
