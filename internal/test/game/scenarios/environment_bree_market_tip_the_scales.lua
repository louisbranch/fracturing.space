local scene = Scenario.new("environment_bree_market_tip_the_scales")

-- Capture the bribe-for-advantage option in the marketplace.
scene:campaign{
  name = "Environment Bree Market Tip the Scales",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A handful of gold buys advantage on a Presence roll.
scene:start_session("Tip the Scales")

-- Gold spend semantics remain unresolved in this fixture.
scene:action_roll{ actor = "Frodo", trait = "presence", difficulty = 10, outcome = "hope", advantage = 1 }
scene:apply_roll_outcome{}

scene:end_session()

return scene
