local scn = Scenario.new("hidden_condition_scouting")
local dh = scn:system("DAGGERHEART")

-- Reflect the scouting example where cover grants the Hidden condition.
scn:campaign{
  name = "Hidden Condition Scouting",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "conditions"
}

scn:pc("Gandalf")

-- Gandalf ducks behind statues without a roll and becomes Hidden.
scn:start_session("Temple Scout")

-- Example: the GM grants Hidden based on cover alone.
dh:apply_condition{ target = "Gandalf", add = { "HIDDEN" }, source = "cover" }

-- Close the session after the condition applies.
scn:end_session()

return scn
