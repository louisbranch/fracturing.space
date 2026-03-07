local scene = Scenario.new("hidden_condition_scouting")
local dh = scene:system("DAGGERHEART")

-- Reflect the scouting example where cover grants the Hidden condition.
scene:campaign{
  name = "Hidden Condition Scouting",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "conditions"
}

scene:pc("Gandalf")

-- Gandalf ducks behind statues without a roll and becomes Hidden.
scene:start_session("Temple Scout")

-- Example: the GM grants Hidden based on cover alone.
dh:apply_condition{ target = "Gandalf", add = { "HIDDEN" }, source = "cover" }

-- Close the session after the condition applies.
scene:end_session()

return scene
