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

-- Example: Presence roll yields gold on success, Stress on failure.
-- Gold payout and stress branch effects remain unresolved.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "fear" }
dh:apply_roll_outcome{}

scn:end_session()

return scn
