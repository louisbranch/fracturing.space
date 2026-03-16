local scn = Scenario.new("adversary_spotlight_cap_relentless")
local dh = scn:system("DAGGERHEART")

scn:campaign{
  name = "Adversary Spotlight Cap Relentless",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "adversary_rules"
}

scn:pc("Frodo")
dh:adversary("Nazgul")

scn:start_session("Relentless Cap")
dh:gm_fear(3)

dh:gm_spend_fear(1):adversary_spotlight("Nazgul", {
  expect_gm_fear_delta = -1,
  expect_gm_move = "spotlight",
  expect_gm_fear_spent = 1
})

dh:gm_spend_fear(1):adversary_spotlight("Nazgul", {
  expect_gm_fear_delta = -1,
  expect_gm_move = "spotlight",
  expect_gm_fear_spent = 1
})

dh:gm_spend_fear(1):adversary_spotlight("Nazgul", {
  expect_error = {
    code = "FAILED_PRECONDITION",
    contains = "spotlight cap reached"
  }
})

dh:expect_gm_fear(1)
scn:end_session()

return scn
