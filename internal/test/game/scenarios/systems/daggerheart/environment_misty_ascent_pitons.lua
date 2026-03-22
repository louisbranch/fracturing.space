local scn = Scenario.new("environment_misty_ascent_pitons")
local dh = scn:system("DAGGERHEART")

-- Capture the pitons rule that trades stress for a failed tick.
scn:campaign{
  name = "Environment Misty Ascent Pitons",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- Pitons let a climber avoid a countdown setback by marking Stress.
scn:start_session("Pitons")

-- Example: on a failed climb, mark Stress instead of ticking up.
-- Partial mapping: failure branch intercept is explicit and avoids countdown setback.
-- Missing DSL: direct stress mark operation for characters.
dh:scene_countdown_create{ name = "Misty Ascent", kind = "progress", current = 0, max = 12, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 12, outcome = "failure_fear" }
dh:apply_roll_outcome{
  on_success = {
    {kind = "scene_countdown_update", name = "Misty Ascent", delta = 1, reason = "climb_progress"},
  },
  on_failure = {
    {kind = "apply_condition", target = "Frodo", add = {"VULNERABLE"}, source = "pitons_stress_tradeoff"},
  },
}

scn:end_session()

return scn
