local scene = Scenario.new("environment_pelennor_battle_adrift")

-- Capture the movement restriction during active battle.
scene:campaign{
  name = "Environment Pelennor Battle Adrift",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- Movement requires agility under fire.
scene:start_session("Adrift on a Sea of Steel")

-- Missing DSL: restrict movement without a successful Agility roll.
scene:action_roll{ actor = "Frodo", trait = "agility", difficulty = 17, outcome = "hope" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
