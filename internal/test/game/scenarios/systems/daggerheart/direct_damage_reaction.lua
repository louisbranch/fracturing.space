local scn = Scenario.new("direct_damage_reaction")
local dh = scn:system("DAGGERHEART")

-- Spotlight a reaction roll against direct damage.
scn:campaign{
  name = "Direct Damage Reaction",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "reaction"
}

scn:pc("Aragorn")

-- An explosive spell threatens Aragorn, calling for a reaction roll.
scn:start_session("Direct Damage")

-- Example: Agility reaction roll vs Difficulty 16 with a forced total to ensure the fixture is deterministic.
dh:reaction_roll{ actor = "Aragorn", trait = "agility", difficulty = 16, total = 19, outcome = "hope" }
dh:apply_reaction_outcome{}

-- Close the session after the reaction.
scn:end_session()

return scn
