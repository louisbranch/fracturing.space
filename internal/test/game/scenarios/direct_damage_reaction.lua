local scene = Scenario.new("direct_damage_reaction")

-- Spotlight a reaction roll against direct damage.
scene:campaign{
  name = "Direct Damage Reaction",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "reaction"
}

scene:pc("Aragorn")

-- An explosive spell threatens Aragorn, calling for a reaction roll.
scene:start_session("Direct Damage")

-- Example: Agility reaction roll vs Difficulty 16 with a forced total to ensure the fixture is deterministic.
scene:reaction_roll{ actor = "Aragorn", trait = "agility", difficulty = 16, total = 19, outcome = "hope" }
scene:apply_reaction_outcome{}

-- Close the session after the reaction.
scene:end_session()

return scene
