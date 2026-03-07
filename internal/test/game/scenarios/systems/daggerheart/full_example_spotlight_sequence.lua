local scn = Scenario.new("full_example_spotlight_sequence")
local dh = scn:system("DAGGERHEART")

-- Follow the example-of-play spotlight order across multiple adversaries.
scn:campaign{
  name = "Full Example Spotlight Sequence",
  system = "DAGGERHEART",
  gm_mode = "HUMAN",
  theme = "spotlight"
}

scn:pc("Sam")
scn:pc("Frodo")
scn:pc("Gandalf")
scn:pc("Aragorn")
dh:adversary("Orc Archer One")
dh:adversary("Orc Archer Two")
dh:adversary("Nazgul")
dh:adversary("Orc Raiders")

-- The GM chains spotlights as threats activate in sequence.
scn:start_session("Spotlight Sequence")
dh:gm_fear(4)

-- Example: archers fire, dredges swarm, then the knight takes center stage.
dh:gm_spend_fear(1):spotlight("Orc Archer One")
dh:gm_spend_fear(1):spotlight("Orc Archer Two")
dh:gm_spend_fear(1):spotlight("Orc Raiders")
dh:gm_spend_fear(1):spotlight("Nazgul")

-- Close the session after the spotlight chain resolves.
scn:end_session()

return scn
