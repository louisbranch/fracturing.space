local scene = Scenario.new("environment_misty_ascent_pitons")

-- Capture the pitons rule that trades stress for a failed tick.
scene:campaign{
  name = "Environment Misty Ascent Pitons",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Pitons let a climber avoid a countdown setback by marking Stress.
scene:start_session("Pitons")

-- Example: on a failed climb, mark Stress instead of ticking up.
-- Partial mapping: failure branch intercept is explicit and avoids countdown setback.
-- Missing DSL: direct stress mark operation for characters.
scene:countdown_create{ name = "Misty Ascent", kind = "progress", current = 0, max = 12, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "agility", difficulty = 12, outcome = "failure_fear" }
scene:apply_roll_outcome{
  on_success = {
    {kind = "countdown_update", name = "Misty Ascent", delta = 1, reason = "climb_progress"},
  },
  on_failure = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "pitons_stress_tradeoff"},
  },
}

scene:end_session()

return scene
