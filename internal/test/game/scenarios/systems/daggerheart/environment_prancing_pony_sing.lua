local scn = Scenario.new("environment_prancing_pony_sing")
local dh = scn:system("DAGGERHEART")

-- Model singing for supper and its stress or reward outcomes.
scn:campaign{
  name = "Environment Prancing Pony Sing",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A performance can earn coin or cost composure.
scn:start_session("Sing for Supper")

-- Presence roll yields gold on success, Stress on failure.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "fear" }
dh:apply_roll_outcome{}

-- On a successful performance the innkeeper pays a handful of gold.
dh:update_gold{
  target = "Frodo",
  handfuls_before = 0,
  handfuls_after = 1,
  bags_before = 0,
  bags_after = 0,
  chests_before = 0,
  chests_after = 0,
  reason = "performance_payout",
}

scn:end_session()

return scn
