local scn = Scenario.new("condition_lifecycle")
local dh = scn:system("DAGGERHEART")

-- Introduce Galadriel so conditions can be applied then cleared.
scn:campaign{
  name = "Condition Lifecycle",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "conditions"
}

scn:pc("Frodo")
dh:adversary("Galadriel")

-- The GM has fear ready to enforce a condition and then clear it.
scn:start_session("Conditions")
dh:gm_fear(3)

-- Galadriel becomes Vulnerable, then the GM spends Fear to frame the break free.
dh:apply_condition{
  target = "Galadriel",
  add = { "VULNERABLE" },
  expect_conditions = { "VULNERABLE" },
  expect_added = { "VULNERABLE" }
}
dh:gm_spend_fear(1):move("custom", {
  description = "Galadriel gathers herself and breaks free.",
  expect_gm_fear_delta = -1,
  expect_gm_move = "custom",
  expect_gm_fear_spent = 1
})
dh:apply_condition{
  target = "Galadriel",
  remove = { "VULNERABLE" },
  source = "break_free",
  expect_conditions = {},
  expect_removed = { "VULNERABLE" }
}

-- Close the session after the condition clears.
scn:end_session()

return scn
