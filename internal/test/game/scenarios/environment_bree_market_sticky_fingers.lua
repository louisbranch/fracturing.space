local scene = Scenario.new("environment_bree_market_sticky_fingers")

-- Capture the Sticky Fingers theft and chase countdowns.
scene:campaign{
  name = "Environment Bree Market Sticky Fingers",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A thief targets a PC, forcing a notice roll and a chase.
scene:start_session("Sticky Fingers")

-- Example: Instinct roll to notice, otherwise trigger progress vs consequence countdowns.
-- Missing DSL: model item loss and chase triggers.
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 10, outcome = "failure_fear" }
scene:countdown_create{ name = "Market Chase", kind = "progress", current = 0, max = 6, direction = "increase" }
scene:countdown_create{ name = "Thief Escape", kind = "consequence", current = 0, max = 4, direction = "increase" }
scene:apply_roll_outcome{
  on_failure_fear = {
    {kind = "countdown_update", name = "Thief Escape", delta = 1, reason = "pickpocket_escape"},
  },
  on_success = {
    {kind = "countdown_update", name = "Market Chase", delta = 1, reason = "spot_thief"},
  },
}

scene:end_session()

return scene
