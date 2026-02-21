local scene = Scenario.new("environment_pelennor_battle_war_magic")

-- Capture large-scale war magic damaging a close area.
scene:campaign{
  name = "Environment Pelennor Battle War Magic",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "environment"
}

scene:pc("Frodo")

-- A mage unleashes battlefield magic.
scene:start_session("War Magic")
scene:gm_fear(1)

-- Missing DSL: apply area reaction roll, damage, and stress on failure.
scene:gm_spend_fear(1):spotlight("Battlefield Nazgul")
scene:reaction_roll{ actor = "Frodo", trait = "agility", difficulty = 17, outcome = "fear" }
scene:apply_reaction_outcome{}

scene:end_session()

return scene
