local scene = Scenario.new("condition_stacking_guard")
local dh = scene:system("DAGGERHEART")

-- Set up Galadriel to test stacking the same condition.
scene:campaign{
  name = "Condition Stacking Guard",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "conditions"
}

scene:pc("Frodo")
dh:adversary("Galadriel")

-- The GM applies conditions and tries to stack the same one twice.
scene:start_session("Condition Guard")

-- Vulnerable is applied, then requested again alongside a new condition.
dh:apply_condition{
  target = "Galadriel",
  add = { "VULNERABLE" },
  source = "spotlight",
  expect_conditions = { "VULNERABLE" },
  expect_added = { "VULNERABLE" }
}
dh:apply_condition{
  target = "Galadriel",
  add = { "VULNERABLE", "HIDDEN" },
  source = "spotlight",
  expect_conditions = { "HIDDEN", "VULNERABLE" },
  expect_added = { "HIDDEN" }
}

-- Close the session after the stacking attempt.
scene:end_session()

return scene
