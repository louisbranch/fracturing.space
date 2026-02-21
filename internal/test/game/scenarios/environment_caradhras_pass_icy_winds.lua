local scene = Scenario.new("environment_caradhras_pass_icy_winds")

-- Model the looping icy winds countdown.
scene:campaign{
  name = "Environment Caradhras Pass Icy Winds",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- The pass inflicts stress at regular intervals.
scene:start_session("Icy Winds")

-- Example: countdown loop 4 triggers Strength reaction or Stress.
-- Stress consequence and loop-reset automation remain unresolved in the fixture DSL.
scene:countdown_create{ name = "Icy Winds", kind = "loop", current = 0, max = 4, direction = "increase" }
scene:countdown_update{ name = "Icy Winds", delta = 4, reason = "loop_trigger" }
scene:reaction_roll{ actor = "Frodo", trait = "strength", difficulty = 15, outcome = "fear", advantage = 1 }

scene:end_session()

return scene
