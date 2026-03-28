local scn = Scenario.new("rejection_gold_no_change")
local dh = scn:system("DAGGERHEART")

-- Updating gold with no denomination changes should be rejected.
scn:campaign{
  name = "Rejection Gold No Change",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rejection"
}

scn:pc("Frodo")

scn:start_session("Rejection")

-- All before/after values are the same — no mutation.
dh:update_gold{
  target = "Frodo",
  handfuls_before = 0,
  handfuls_after = 0,
  bags_before = 0,
  bags_after = 0,
  chests_before = 0,
  chests_after = 0,
  reason = "no_change",
  expect_error = {code = "INTERNAL"}
}

scn:end_session()
return scn
