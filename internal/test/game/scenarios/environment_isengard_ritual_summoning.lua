local scene = Scenario.new("environment_isengard_ritual_summoning")

-- Capture the summoning countdown that triggers on Fear rolls.
scene:campaign{
  name = "Environment Isengard Ritual Summoning",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")
scene:adversary("Saruman")
scene:adversary("Shadow Wraith")

-- The ritual builds toward summoning a demon.
scene:start_session("Summoning")

-- Example: countdown ticks down when PCs roll with Fear.
-- Summon-on-trigger spawn behavior remains unresolved in this fixture.
scene:countdown_create{ name = "Summon Demon", kind = "consequence", current = 0, max = 6, direction = "increase" }
scene:action_roll{ actor = "Frodo", trait = "instinct", difficulty = 14, outcome = "fear" }
scene:countdown_update{ name = "Summon Demon", delta = 1, reason = "fear_outcome" }

scene:end_session()

return scene
