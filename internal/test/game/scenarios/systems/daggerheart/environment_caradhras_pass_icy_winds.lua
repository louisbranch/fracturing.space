local scn = Scenario.new("environment_caradhras_pass_icy_winds")
local dh = scn:system("DAGGERHEART")

-- Model the looping icy winds countdown.
scn:campaign{
  name = "Environment Caradhras Pass Icy Winds",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The pass inflicts stress at regular intervals.
scn:start_session("Icy Winds")

-- Example: countdown loop 4 triggers Strength reaction or Stress.
-- Stress consequence and loop-reset automation remain unresolved in the fixture DSL.
dh:scene_countdown_create{ name = "Icy Winds", kind = "loop", current = 0, max = 4, direction = "increase" }
dh:scene_countdown_update{ name = "Icy Winds", delta = 4, reason = "loop_trigger" }
dh:reaction_roll{ actor = "Frodo", trait = "strength", difficulty = 15, outcome = "fear", advantage = 1 }

scn:end_session()

return scn
