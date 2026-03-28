local scn = Scenario.new("rejection_condition_remove_missing")
local dh = scn:system("DAGGERHEART")

-- Removing a condition the adversary does not have should be rejected.
scn:campaign{
  name = "Rejection Condition Remove Missing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rejection"
}

scn:pc("Scout")
dh:adversary("Goblin")

scn:start_session("Rejection")

-- Goblin has no conditions.  Removing VULNERABLE should fail.
dh:apply_condition{
  target = "Goblin",
  remove = { "VULNERABLE" },
  expect_error = {code = "FAILED_PRECONDITION"}
}

scn:end_session()
return scn
