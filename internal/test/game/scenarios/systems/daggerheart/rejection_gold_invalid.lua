local scn = Scenario.new("rejection_gold_invalid")
local dh = scn:system("DAGGERHEART")

-- Gold denominations are constrained: handfuls 0-9, bags 0-9, chests 0-1.
-- Setting values beyond these limits should be rejected.
scn:campaign{
  name = "Rejection Gold Invalid",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rejection"
}

scn:pc("Frodo")

scn:start_session("Rejection")

-- Try to set handfuls beyond the valid range (max 9).
dh:update_gold{
  target = "Frodo",
  handfuls_before = 0,
  handfuls_after = 10,
  bags_before = 0,
  bags_after = 0,
  chests_before = 0,
  chests_after = 0,
  reason = "too_much_gold",
  expect_error = {code = "INTERNAL"}
}

scn:end_session()
return scn
