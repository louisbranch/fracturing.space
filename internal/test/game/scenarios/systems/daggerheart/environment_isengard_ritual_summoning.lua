local scn = Scenario.new("environment_isengard_ritual_summoning")
local dh = scn:system("DAGGERHEART")

-- Capture the summoning countdown that triggers on Fear rolls.
scn:campaign{
  name = "Environment Isengard Ritual Summoning",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")
dh:adversary("Saruman")
dh:adversary("Shadow Wraith")

-- The ritual builds toward summoning a demon.
scn:start_session("Summoning")

-- Example: countdown ticks down when PCs roll with Fear.
-- Summon-on-trigger spawn behavior remains unresolved in this fixture.
dh:scene_countdown_create{ name = "Summon Demon", kind = "consequence", current = 0, max = 6, direction = "increase" }
dh:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 14, outcome = "fear" }
dh:scene_countdown_update{ name = "Summon Demon", delta = 1, reason = "fear_outcome" }

scn:end_session()

return scn
