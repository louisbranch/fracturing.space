-- Scenario: participant + character chaining
local scn = Scenario.new("participant_chain")
local dh = scn:system("DAGGERHEART")

-- Create campaign
scn:campaign({
  name = "Participant Chain",
  system = "DAGGERHEART",
})

-- Create participant and character in one chain
scn:participant({name = "John"}):character({name = "Frodo"})

return scn
