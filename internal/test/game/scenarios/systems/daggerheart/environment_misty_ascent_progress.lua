local scn = Scenario.new("environment_misty_ascent_progress")
local dh = scn:system("DAGGERHEART")

-- Model the Misty Ascent progress countdown and roll outcomes.
scn:campaign{
  name = "Environment Misty Ascent Progress",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- The party climbs using a progress countdown.
scn:start_session("Misty Ascent")

-- Example: Progress Countdown (12) ticks based on roll outcomes.
-- Partial mapping: tiered countdown updates are explicit by roll branch.
-- Missing DSL: branch-level no-op steps for failure tiers with no advancement.
dh:countdown_create{ name = "Misty Ascent", kind = "progress", current = 0, max = 12, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "agility", difficulty = 12, outcome = "success_hope" }
dh:apply_roll_outcome{
  on_critical = {
    {kind = "countdown_update", name = "Misty Ascent", delta = 3, reason = "critical_progress"},
  },
  on_success_hope = {
    {kind = "countdown_update", name = "Misty Ascent", delta = 2, reason = "hopeful_progress"},
  },
  on_success_fear = {
    {kind = "countdown_update", name = "Misty Ascent", delta = 1, reason = "tense_progress"},
  },
}

scn:end_session()

return scn
