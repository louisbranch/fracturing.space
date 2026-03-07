local scn = Scenario.new("environment_waylaid_relative_strength")
local dh = scn:system("DAGGERHEART")

-- Model Waylaid using the highest adversary Difficulty.
scn:campaign{
  name = "Environment Waylaid Relative Strength",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Orc Sniper")
dh:adversary("Orc Lackey")

-- The ambushed difficulty matches the toughest adversary.
scn:start_session("Relative Strength")

-- Explicit anchor: highest adversary difficulty is represented by Orc Sniper.
dh:adversary_update{ target = "Orc Sniper", evasion = 15, notes = "relative_strength_anchor" }
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 15, outcome = "fear" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
