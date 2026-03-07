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

-- Spend gold to bribe the merchant for advantage.
dh:update_gold{
  target = "Frodo",
  handfuls_before = 2,
  handfuls_after = 1,
  bags_before = 0,
  bags_after = 0,
  chests_before = 0,
  chests_after = 0,
  reason = "bribe_merchant",
}

-- Now roll with advantage from the bribe.
dh:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "hope", advantage = 1 }
dh:apply_roll_outcome{}

scn:end_session()

return scn
