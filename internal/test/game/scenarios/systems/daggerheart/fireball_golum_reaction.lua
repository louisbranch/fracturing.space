local scene = Scenario.new("fireball_golum_reaction")
local dh = scene:system("DAGGERHEART")

-- Reflect the fireball versus Golum reaction roll example.
scene:campaign{
  name = "Fireball Golum Reaction",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "battle"
}

scene:pc("Gandalf")
dh:adversary("Golum")

-- Gandalf unleashes fire while the GM boosts the thief's reaction.
scene:start_session("Fireball Chase")
dh:gm_fear(1)

dh:attack{
  actor = "Gandalf",
  target = "Golum",
  trait = "spellcast",
  difficulty = 0,
  outcome = "hope",
  damage_type = "magic"
}

-- Example: the GM spends fear to add +3 to the thief's reaction roll.
-- Missing DSL: adversary reaction roll with an experience bonus.
dh:gm_spend_fear(1):spotlight("Golum")

-- Close the session after the reaction example.
scene:end_session()

return scene
