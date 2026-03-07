local scn = Scenario.new("advantage_cancellation")
local dh = scn:system("DAGGERHEART")

-- Model advantage/disadvantage cancellation (two up, one down).
scn:campaign{
  name = "Advantage Cancellation",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "advantage"
}

scn:pc("Frodo")

-- Two sources of advantage and one of disadvantage resolve to advantage.
scn:start_session("Advantage Cancellation")

dh:action_roll{
  actor = "Frodo",
  trait = "presence",
  difficulty = 12,
  advantage = 2,
  disadvantage = 1,
  outcome = "hope"
}

scn:end_session()

return scn
