local scene = Scenario.new("environment_rivendell_sanctuary_relentless_hope")

-- Model the once-per-scene stress spend to flip Fear to Hope.
scene:campaign{
  name = "Environment Rivendell Sanctuary Relentless Hope",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A PC marks Stress to turn Fear into Hope once per scene.
scene:start_session("Relentless Hope")

-- Missing DSL: convert a Fear outcome to Hope and mark Stress.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 13, outcome = "failure_fear" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
