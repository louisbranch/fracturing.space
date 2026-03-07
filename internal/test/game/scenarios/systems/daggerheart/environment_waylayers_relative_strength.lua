local scn = Scenario.new("environment_waylayers_relative_strength")
local dh = scn:system("DAGGERHEART")

-- Model Orc Waylayers using the highest adversary Difficulty.
scn:campaign{
  name = "Environment Orc Waylayers Relative Strength",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Orc Sniper")
dh:adversary("Orc Lackey")

-- The ambush difficulty matches the toughest adversary.
scn:start_session("Relative Strength")

-- Explicit anchor: highest adversary difficulty is represented by Orc Sniper.
dh:adversary_update{ target = "Orc Sniper", evasion = 15, notes = "relative_strength_anchor" }
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 15, outcome = "hope" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
