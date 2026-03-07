local scene = Scenario.new("environment_waylaid_relative_strength")
local dh = scene:system("DAGGERHEART")

-- Model Waylaid using the highest adversary Difficulty.
scene:campaign{
  name = "Environment Waylaid Relative Strength",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
dh:adversary("Orc Sniper")
dh:adversary("Orc Lackey")

-- The ambushed difficulty matches the toughest adversary.
scene:start_session("Relative Strength")

-- Explicit anchor: highest adversary difficulty is represented by Orc Sniper.
dh:adversary_update{ target = "Orc Sniper", evasion = 15, notes = "relative_strength_anchor" }
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 15, outcome = "fear" }
dh:apply_roll_outcome{}

scene:end_session()

return scene
