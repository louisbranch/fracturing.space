local scn = Scenario.new("environment_pelennor_battle_war_magic")
local dh = scn:system("DAGGERHEART")

-- Capture large-scale war magic damaging a close area.
scn:campaign{
  name = "Environment Pelennor Battle War Magic",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scn:pc("Frodo")

-- A mage unleashes battlefield magic.
scn:start_session("War Magic")
dh:gm_fear(1)

-- Missing DSL: apply area reaction roll, damage, and stress on failure.
dh:gm_spend_fear(1):spotlight("Battlefield Nazgul")
dh:reaction_roll{ actor = "Frodo", trait = "agility", difficulty = 17, outcome = "fear" }
dh:apply_reaction_outcome{}

scn:end_session()

return scn
