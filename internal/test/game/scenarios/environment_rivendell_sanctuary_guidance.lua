local scene = Scenario.new("environment_rivendell_sanctuary_guidance")

-- Capture divine guidance outcomes from prayer.
scene:campaign{
  name = "Environment Rivendell Sanctuary Guidance",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A prayer triggers a roll for guidance.
scene:start_session("Divine Guidance")

-- Missing DSL: apply outcome-based clarity and Hope gain.
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 13, outcome = "hope" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
