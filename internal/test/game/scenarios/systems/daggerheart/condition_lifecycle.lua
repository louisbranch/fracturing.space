local scene = Scenario.new("condition_lifecycle")
local dh = scene:system("DAGGERHEART")

-- Introduce Galadriel so conditions can be applied then cleared.
scene:campaign{
  name = "Condition Lifecycle",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "conditions"
}

scene:pc("Frodo")
dh:adversary("Galadriel")

-- The GM has fear ready to enforce a condition and then clear it.
scene:start_session("Conditions")
dh:gm_fear(3)

-- Galadriel becomes Vulnerable, then uses a spotlight moment to break free.
dh:apply_condition{
  target = "Galadriel",
  add = { "VULNERABLE" },
  expect_conditions = { "VULNERABLE" },
  expect_added = { "VULNERABLE" }
}
dh:gm_spend_fear(1):spotlight("Galadriel", { expect_gm_fear_delta = -1, expect_gm_move = "spotlight", expect_gm_fear_spent = 1 })
dh:apply_condition{
  target = "Galadriel",
  remove = { "VULNERABLE" },
  source = "break_free",
  expect_conditions = {},
  expect_removed = { "VULNERABLE" }
}

-- Close the session after the condition clears.
scene:end_session()

return scene
