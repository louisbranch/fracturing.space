local scene = Scenario.new("environment_helms_deep_siege_secret_entrance")

-- Capture the secret entrance discovery roll.
scene:campaign{
  name = "Environment Helms Deep Siege Secret Entrance",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A PC searches for a hidden passage.
scene:start_session("Secret Entrance")

-- Missing DSL: reveal a secret route with Instinct/Knowledge success.
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 17, outcome = "hope" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
