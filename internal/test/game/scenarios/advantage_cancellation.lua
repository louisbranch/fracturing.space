local scene = Scenario.new("advantage_cancellation")

-- Model advantage/disadvantage cancellation (two up, one down).
scene:campaign{
  name = "Advantage Cancellation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "advantage"
}

scene:pc("Frodo")

-- Two sources of advantage and one of disadvantage resolve to advantage.
scene:start_session("Advantage Cancellation")

scene:action_roll{
  actor = "Frodo",
  trait = "presence",
  difficulty = 12,
  advantage = 2,
  disadvantage = 1,
  outcome = "hope"
}

scene:end_session()

return scene
