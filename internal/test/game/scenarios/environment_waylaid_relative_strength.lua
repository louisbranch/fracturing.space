local scene = Scenario.new("environment_waylaid_relative_strength")

-- Model Waylaid using the highest adversary Difficulty.
scene:campaign{
  name = "Environment Waylaid Relative Strength",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Orc Sniper")
scene:adversary("Orc Lackey")

-- The ambushed difficulty matches the toughest adversary.
scene:start_session("Relative Strength")

-- Explicit anchor: highest adversary difficulty is represented by Orc Sniper.
scene:adversary_update{ target = "Orc Sniper", evasion = 15, notes = "relative_strength_anchor" }
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 15, outcome = "fear" }
scene:apply_roll_outcome{}

scene:end_session()

return scene
