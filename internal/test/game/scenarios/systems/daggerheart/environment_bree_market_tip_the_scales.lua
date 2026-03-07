local scn = Scenario.new("environment_bree_market_tip_the_scales")
local dh = scn:system("DAGGERHEART")

-- Capture the bribe-for-advantage option in the marketplace.
scn:campaign{
  name = "Environment Bree Market Tip the Scales",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A handful of gold buys advantage on a Presence roll.
scn:start_session("Tip the Scales")

-- Gold spend semantics remain unresolved in this fixture.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "hope", advantage = 1 }
dh:apply_roll_outcome{}

scn:end_session()

return scn
