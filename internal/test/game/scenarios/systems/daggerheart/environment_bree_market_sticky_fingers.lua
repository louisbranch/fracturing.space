local scn = Scenario.new("environment_bree_market_sticky_fingers")
local dh = scn:system("DAGGERHEART")

-- Capture the Sticky Fingers theft and chase countdowns.
scn:campaign{
  name = "Environment Bree Market Sticky Fingers",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A thief targets a PC, forcing a notice roll and a chase.
scn:start_session("Sticky Fingers")

-- Example: Instinct roll to notice, otherwise trigger progress vs consequence countdowns.
-- Missing DSL: model item loss and chase triggers.
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 10, outcome = "failure_fear" }
dh:countdown_create{ name = "Market Chase", kind = "progress", current = 0, max = 6, direction = "increase" }
dh:countdown_create{ name = "Thief Escape", kind = "consequence", current = 0, max = 4, direction = "increase" }
dh:apply_roll_outcome{
  on_failure_fear = {
    {kind = "countdown_update", name = "Thief Escape", delta = 1, reason = "pickpocket_escape"},
  },
  on_success = {
    {kind = "countdown_update", name = "Market Chase", delta = 1, reason = "spot_thief"},
  },
}

scn:end_session()

return scn
