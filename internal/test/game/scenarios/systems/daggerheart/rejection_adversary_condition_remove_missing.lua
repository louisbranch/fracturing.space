local scn = Scenario.new("rejection_adversary_condition_remove_missing")
local dh = scn:system("DAGGERHEART")

-- Removing a condition from a PC that does not have it should be rejected.
scn:campaign{
  name = "Rejection PC Condition Remove Missing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rejection"
}

scn:pc("Frodo")

scn:start_session("Rejection")

-- Frodo has no conditions.  Removing VULNERABLE should fail.
dh:apply_condition{
  target = "Frodo",
  remove = { "VULNERABLE" },
  expect_error = {code = "FAILED_PRECONDITION"}
}

scn:end_session()
return scn
