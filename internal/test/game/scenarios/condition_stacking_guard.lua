local scene = Scenario.new("condition_stacking_guard")

-- Set up Galadriel to test stacking the same condition.
scene:campaign{
  name = "Condition Stacking Guard",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "conditions"
}

scene:pc("Frodo")
scene:npc("Galadriel")

-- The GM applies conditions and tries to stack the same one twice.
scene:start_session("Condition Guard")

-- Vulnerable is applied, then requested again alongside a new condition.
-- Missing DSL: apply conditions to adversaries; Galadriel stands in.
-- Missing DSL: assert the duplicate is ignored while the new one sticks.
scene:apply_condition{ target = "Galadriel", add = { "VULNERABLE" }, source = "spotlight" }
scene:apply_condition{ target = "Galadriel", add = { "VULNERABLE", "HIDDEN" }, source = "spotlight" }

-- Close the session after the stacking attempt.
scene:end_session()

return scene
