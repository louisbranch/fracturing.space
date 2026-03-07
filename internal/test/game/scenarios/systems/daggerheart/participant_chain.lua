-- Scenario: participant + character chaining
local scene = Scenario.new("participant_chain")
local dh = scene:system("DAGGERHEART")

-- Create campaign
scene:campaign({
  name = "Participant Chain",
  system = "DAGGERHEART",
})

-- Create participant and character in one chain
scene:participant({name = "John"}):character({name = "Frodo"})

return scene
