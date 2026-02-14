local scene = Scenario.new("condition_lifecycle")

-- Introduce Galadriel so conditions can be applied then cleared.
scene:campaign{
  name = "Condition Lifecycle",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "conditions"
}

scene:pc("Frodo")
scene:npc("Galadriel")

-- The GM has fear ready to enforce a condition and then clear it.
scene:start_session("Conditions")
scene:gm_fear(3)

-- Galadriel becomes Vulnerable, then uses a spotlight moment to break free.
-- Missing DSL: apply conditions to adversaries; Galadriel stands in.
scene:apply_condition{ target = "Galadriel", add = { "VULNERABLE" } }
scene:gm_spend_fear(1):spotlight("Galadriel")
scene:apply_condition{ target = "Galadriel", remove = { "VULNERABLE" }, source = "break_free" }

-- Close the session after the condition clears.
scene:end_session()

return scene
