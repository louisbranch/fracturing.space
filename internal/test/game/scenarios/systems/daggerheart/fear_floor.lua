local scn = Scenario.new("fear_floor")
local dh = scn:system("DAGGERHEART")

-- Set a simple scene to drain the fear pool to zero.
scn:campaign{
  name = "Fear Floor",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "fear"
}

scn:pc("Frodo")
dh:adversary("Nazgul")

-- The GM spends fear until the pool runs dry.
scn:start_session("Fear Floor")
dh:gm_fear(2)

-- Two spends should bring fear to zero without going negative.
dh:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
dh:gm_spend_fear(1):spotlight("Nazgul", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
dh:gm_fear(0)

-- Close the session once fear hits the floor.
scn:end_session()

return scn
