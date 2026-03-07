local scn = Scenario.new("difficulty_setting_examples")
local dh = scn:system("DAGGERHEART")

-- Sample a few difficulty settings across common actions.
scn:campaign{
  name = "Difficulty Setting Examples",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "difficulty"
}

scn:pc("Frodo")

-- The GM calls for rolls with escalating difficulty.
scn:start_session("Difficulty Settings")

-- Example: Agility difficulty 5 to sprint within Close range.
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 5, outcome = "hope" }

-- Example: Finesse difficulty 15 to ride through rough terrain.
dh:action_roll{ actor = "Frodo", trait = "finesse", difficulty = 15, outcome = "hope" }

-- Example: Strength difficulty 25 to lift a large beast.
dh:action_roll{ actor = "Frodo", trait = "strength", difficulty = 25, outcome = "fear" }

-- Close the session after the difficulty samples.
scn:end_session()

return scn
