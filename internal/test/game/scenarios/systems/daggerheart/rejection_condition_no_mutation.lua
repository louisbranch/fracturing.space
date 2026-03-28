local scn = Scenario.new("rejection_condition_no_mutation")
local dh = scn:system("DAGGERHEART")

-- Adding a condition the target already has should be rejected as a no-op.
scn:campaign{
  name = "Rejection Condition No Mutation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "rejection"
}

scn:pc("Scout")
dh:adversary("Goblin")

scn:start_session("Rejection")

-- Apply VULNERABLE once — should succeed.
dh:apply_condition{
  target = "Goblin",
  add = { "VULNERABLE" },
  expect_conditions = { "VULNERABLE" },
  expect_added = { "VULNERABLE" }
}

-- Apply VULNERABLE again with no other changes — rejected as no mutation.
dh:apply_condition{
  target = "Goblin",
  add = { "VULNERABLE" },
  expect_error = {code = "FAILED_PRECONDITION"}
}

scn:end_session()
return scn
