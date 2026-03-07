local scn = Scenario.new("progress_countdown_climb")
local dh = scn:system("DAGGERHEART")

-- Model the mountain ascent progress countdown from the example of play.
scn:campaign{
  name = "Progress Countdown Climb",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "countdown"
}

scn:pc("Frodo")
scn:pc("Sam")

-- The GM sets a shorter progress countdown due to helpful guidance.
scn:start_session("Whitecrest Ascent")

-- Example: progress countdown starts at 3 instead of 5.
dh:countdown_create{ name = "Whitecrest Ascent", kind = "progress", current = 3, max = 3, direction = "decrease" }

-- Sam succeeds with Fear, advancing the climb despite consequences.
-- Partial mapping: dynamic tier-based countdown updates are explicit.
-- Missing DSL: branch-level no-op steps for failure tiers with no advancement.
dh:action_roll{ actor = "Sam", trait = "agility", difficulty = 12, outcome = "success_fear" }
dh:apply_roll_outcome{
  on_critical = {
    {kind = "countdown_update", name = "Whitecrest Ascent", delta = -3, reason = "critical_ascent"},
  },
  on_success_hope = {
    {kind = "countdown_update", name = "Whitecrest Ascent", delta = -2, reason = "strong_ascent"},
  },
  on_success_fear = {
    {kind = "countdown_update", name = "Whitecrest Ascent", delta = -1, reason = "steady_ascent"},
  },
}

-- Close the session after the ascent advances.
scn:end_session()

return scn
